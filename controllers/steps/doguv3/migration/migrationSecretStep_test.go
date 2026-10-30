package migration

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	return scheme
}

func sensitiveConfig(configYaml string) *corev1.Secret {
	return &corev1.Secret{
		Name:      "nexus-config",
		Namespace: "ecosystem",
		Labels:    map[string]string{"k8s.cloudogu.com/type": "sensitive-config"},
		Data:      map[string][]byte{"config.yaml": []byte(configYaml)},
	}
}

func getMigrationSecret(t *testing.T, k8sClient client.Client) *corev1.Secret {
	t.Helper()
	secret := &corev1.Secret{}
	require.NoError(t, k8sClient.Get(testCtx, client.ObjectKey{Namespace: "ecosystem", Name: "nexus-migration"}, secret))
	return secret
}

func TestMigrationSecretStep_Run(t *testing.T) {
	t.Run("creates the migration secret from the service accounts of the sensitive config", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithReturnManagedFields().WithObjects(sensitiveConfig(`
sa-postgresql:
  username: nexus_uiuxsp
  password: secret1
sa-cas:
  username: nexus
admin_password: secret2
`)).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" created`)

		result := NewMigrationSecretStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		secret := getMigrationSecret(t, k8sClient)
		assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
		assert.Equal(t, map[string][]byte{
			"postgresql_username": []byte("nexus_uiuxsp"),
			"postgresql_password": []byte("secret1"),
			"cas_username":        []byte("nexus"),
		}, secret.Data)
		assert.Equal(t, map[string]string{
			"k8s.cloudogu.com/app":       "ces",
			"k8s.cloudogu.com/dogu.name": "nexus",
			"k8s.cloudogu.com/type":      "migration",
		}, secret.Labels)
		require.Len(t, secret.ManagedFields, 1)
		assert.Equal(t, "k8s-dogu-operator-migration-secret-step", secret.ManagedFields[0].Manager)
		assert.Equal(t, metav1.ManagedFieldsOperationApply, secret.ManagedFields[0].Operation)
	})

	t.Run("updates the migration secret and removes entries gone from the sensitive config", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(sensitiveConfig(`
sa-postgresql:
  username: nexus
  password: old
`)).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" created`)
		step := NewMigrationSecretStep(k8sClient, recorder)
		require.Equal(t, stepsv3.Continue(), step.Run(testCtx, dogu))

		require.NoError(t, k8sClient.Update(testCtx, sensitiveConfig("sa-postgresql: {password: new}")))
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" updated`)

		result := step.Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Equal(t, map[string][]byte{"postgresql_password": []byte("new")}, getMigrationSecret(t, k8sClient).Data)
	})

	t.Run("empties the migration secret if no service accounts are left", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(sensitiveConfig("sa-postgresql: {username: nexus}")).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" created`)
		step := NewMigrationSecretStep(k8sClient, recorder)
		require.Equal(t, stepsv3.Continue(), step.Run(testCtx, dogu))

		require.NoError(t, k8sClient.Update(testCtx, sensitiveConfig("admin_password: secret")))
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" updated`)

		result := step.Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Empty(t, getMigrationSecret(t, k8sClient).Data)
	})

	t.Run("records no event if the sensitive config is unchanged", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(sensitiveConfig("sa-postgresql: {username: nexus}")).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration secret "nexus-migration" created`).Once()
		step := NewMigrationSecretStep(k8sClient, recorder)
		require.Equal(t, stepsv3.Continue(), step.Run(testCtx, dogu))

		result := step.Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Equal(t, map[string][]byte{"postgresql_username": []byte("nexus")}, getMigrationSecret(t, k8sClient).Data)
	})

	t.Run("aborts with a warning and keeps the migration secret if service account entries collide", func(t *testing.T) {
		dogu := testDogu()
		existing := &corev1.Secret{
			Name: "nexus-migration", Namespace: "ecosystem",
			Data: map[string][]byte{"ldap_bind_password": []byte("migrated")},
		}
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(existing, sensitiveConfig(`
sa-ldap_bind:
  password: secret1
sa-ldap:
  bind_password: secret2
`)).Build()
		recorder := NewMockEventRecorder(t)
		msg := `sensitive config paths "sa-ldap.bind_password" and "sa-ldap_bind.password" both result in the migration secret key "ldap_bind_password"`

		result := NewMigrationSecretStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Abort(v3beta1.ReasonMigrationFailed, msg), result)
		assert.Equal(t, map[string][]byte{"ldap_bind_password": []byte("migrated")}, getMigrationSecret(t, k8sClient).Data)
	})

	t.Run("aborts with a warning if the sensitive config cannot be parsed", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(sensitiveConfig("sa-postgresql: [unclosed")).Build()
		recorder := NewMockEventRecorder(t)

		result := NewMigrationSecretStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
		assert.Zero(t, result.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonMigrationFailed, result.ReadyReason)
	})

	t.Run("requeues with a warning if the migration secret cannot be applied", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(sensitiveConfig("admin_password: secret")).WithInterceptorFuncs(interceptor.Funcs{
			Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
				return assert.AnError
			},
		}).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, `failed to apply migration secret "nexus-migration": assert.AnError general error for testing`)

		result := NewMigrationSecretStep(k8sClient, recorder).Run(testCtx, dogu)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
		assert.False(t, result.Continue)
	})

	t.Run("keeps the migration secret untouched if the sensitive config is gone", func(t *testing.T) {
		dogu := testDogu()
		existing := &corev1.Secret{
			Name: "nexus-migration", Namespace: "ecosystem",
			Data: map[string][]byte{"postgresql_username": []byte("migrated")},
		}
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(existing).Build()

		result := NewMigrationSecretStep(k8sClient, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Equal(t, map[string][]byte{"postgresql_username": []byte("migrated")}, getMigrationSecret(t, k8sClient).Data)
	})

	t.Run("ignores a config secret which is no sensitive config", func(t *testing.T) {
		config := sensitiveConfig("admin_password: secret")
		config.Labels = map[string]string{"k8s.cloudogu.com/type": "dogu-config"}
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(config).Build()

		result := NewMigrationSecretStep(k8sClient, NewMockEventRecorder(t)).Run(testCtx, testDogu())

		assert.Equal(t, stepsv3.Continue(), result)
		err := k8sClient.Get(testCtx, client.ObjectKey{Namespace: "ecosystem", Name: "nexus-migration"}, &corev1.Secret{})
		assert.True(t, apierrors.IsNotFound(err))
	})

	t.Run("creates no migration secret without sensitive config", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()

		result := NewMigrationSecretStep(k8sClient, NewMockEventRecorder(t)).Run(testCtx, testDogu())

		assert.Equal(t, stepsv3.Continue(), result)
		err := k8sClient.Get(testCtx, client.ObjectKey{Namespace: "ecosystem", Name: "nexus-migration"}, &corev1.Secret{})
		assert.True(t, apierrors.IsNotFound(err))
	})
}
