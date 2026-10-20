package migration

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "ecosystem"

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	return scheme
}

func v3Dogu(name string) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec:       v3beta1.DoguSpec{Name: name, DoguNamespace: "official", DoguApiVersion: v3beta1.DoguApiVersionV3},
	}
}

func detect(t *testing.T, dogu *v3beta1.Dogu, objects ...client.Object) (State, error) {
	t.Helper()
	cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...).Build()
	return NewDetector(cl).Detect(context.Background(), dogu)
}

func TestDetector_Detect(t *testing.T) {
	t.Run("v3 dogu without any resources needs no migration", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"))

		require.NoError(t, err)
		assert.Equal(t, NotRequired, state)
	})

	t.Run("sensitive config of the v2 dogu requires a migration", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), sensitiveConfig("nexus"))

		require.NoError(t, err)
		assert.Equal(t, Required, state)
	})

	t.Run("data volume claim of the v2 dogu requires a migration", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), pvc("nexus", nil))

		require.NoError(t, err)
		assert.Equal(t, Required, state)
	})

	t.Run("volume claim managed by helm belongs to the v3 dogu", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), pvc("nexus", map[string]string{"app.kubernetes.io/managed-by": "Helm"}))

		require.NoError(t, err)
		assert.Equal(t, NotRequired, state)
	})

	t.Run("migration secret alone means the migration is in progress", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "nexus-migration", Namespace: testNamespace}})

		require.NoError(t, err)
		assert.Equal(t, InProgress, state)
	})

	t.Run("migration volume claim alone means the migration is in progress", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), pvc("nexus-migration", nil))

		require.NoError(t, err)
		assert.Equal(t, InProgress, state)
	})

	t.Run("v2 resources next to migration resources still require the migration", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), sensitiveConfig("nexus"), pvc("nexus-migration", nil))

		require.NoError(t, err)
		assert.Equal(t, Required, state)
	})

	t.Run("v2 dogu is never migrated", func(t *testing.T) {
		dogu := v3Dogu("nexus")
		dogu.Spec.DoguApiVersion = v3beta1.DoguApiVersionV2

		state, err := detect(t, dogu, sensitiveConfig("nexus"), pvc("nexus", nil))

		require.NoError(t, err)
		assert.Equal(t, NotRequired, state)
	})

	t.Run("secret named like the config but without sensitive config label is no v2 resource", func(t *testing.T) {
		state, err := detect(t, v3Dogu("nexus"), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "nexus-config", Namespace: testNamespace}})

		require.NoError(t, err)
		assert.Equal(t, NotRequired, state)
	})

	t.Run("propagates errors other than not found", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return errors.New("api down")
			},
		}).Build()

		_, err := NewDetector(cl).Detect(context.Background(), v3Dogu("nexus"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "ecosystem/nexus-config")
		assert.ErrorContains(t, err, "api down")
	})
}

func pvc(name string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace, Labels: labels}}
}

func sensitiveConfig(doguName string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      doguName + "-config",
		Namespace: testNamespace,
		Labels:    map[string]string{"app": "ces", "dogu.name": doguName, "k8s.cloudogu.com/type": "sensitive-config"},
	}}
}
