package migration

import (
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func migrationSecret() *corev1.Secret {
	return &corev1.Secret{
		Name: "nexus-migration", Namespace: "ecosystem",
		Data: map[string][]byte{"postgresql_username": []byte("nexus"), "postgresql_password": []byte("secret")},
	}
}

func migrationPVC() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{Name: "nexus-migration", Namespace: "ecosystem"}
}

// newClient builds a fake client with a deduced type converter, because the default one fails to apply network policies.
func newClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().WithScheme(testScheme(t)).WithTypeConverters(managedfields.NewDeducedTypeConverter()).WithObjects(objects...).Build()
}

func listNetworkPolicies(t *testing.T, k8sClient client.Client) []networkingv1.NetworkPolicy {
	t.Helper()
	policies := &networkingv1.NetworkPolicyList{}
	require.NoError(t, k8sClient.List(testCtx, policies, client.InNamespace("ecosystem")))

	return policies.Items
}

func TestPrepareMigrationJobsStep_Run(t *testing.T) {
	t.Run("creates a network policy per producer of the migration secret", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := newClient(t, migrationSecret())
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, `Migration network policy "nexus-migration-postgresql" created`)

		result := NewPrepareMigrationJobsStep(k8sClient, charts, recorder, true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		policies := listNetworkPolicies(t, k8sClient)
		require.Len(t, policies, 1)
		assert.Equal(t, "nexus-migration-postgresql", policies[0].Name)
	})

	t.Run("records no event for an existing network policy", func(t *testing.T) {
		dogu := testDogu()
		existing := &networkingv1.NetworkPolicy{Name: "nexus-migration-postgresql", Namespace: "ecosystem"}
		k8sClient := newClient(t, migrationSecret(), existing)
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		recorder := NewMockEventRecorder(t)

		result := NewPrepareMigrationJobsStep(k8sClient, charts, recorder, true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Empty(t, recorder.Calls)
	})

	t.Run("creates no network policies when they are disabled", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := newClient(t, migrationSecret())
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)

		result := NewPrepareMigrationJobsStep(k8sClient, charts, NewMockEventRecorder(t), false).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Empty(t, listNetworkPolicies(t, k8sClient))
	})

	t.Run("continues for a chart without migration jobs and without migration data", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := newClient(t, migrationSecret())
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(false, nil)

		result := NewPrepareMigrationJobsStep(k8sClient, charts, NewMockEventRecorder(t), true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
		assert.Empty(t, listNetworkPolicies(t, k8sClient))
	})

	t.Run("aborts for a chart without migration jobs, because the migration data would be lost", func(t *testing.T) {
		dogu := testDogu()
		k8sClient := newClient(t, migrationPVC())
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(false, nil)

		result := NewPrepareMigrationJobsStep(k8sClient, charts, NewMockEventRecorder(t), true).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Abort(v3beta1.ReasonMigrationFailed, `the chart ships no migration job, so the data of the migration volume claim "nexus-migration" would be lost`), result)
	})

	t.Run("chart error requeues with a warning", func(t *testing.T) {
		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(false, assert.AnError)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, fmt.Sprintf("failed to check if chart %q has migration job: %v", dogu.Name, assert.AnError))

		result := NewPrepareMigrationJobsStep(newClient(t), charts, recorder, true).Run(testCtx, dogu)

		assert.False(t, result.Continue)
		assert.ErrorContains(t, result.Err, assert.AnError.Error())
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})
}
