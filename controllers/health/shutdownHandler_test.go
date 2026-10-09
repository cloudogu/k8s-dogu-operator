package health

import (
	"context"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const shutdownTestNamespace = "ecosystem"

func shutdownTestClient(t *testing.T, funcs interceptor.Funcs, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v2.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&v2.Dogu{}, &v3beta1.Dogu{}).
		WithInterceptorFuncs(funcs).
		Build()
}

func v2ShutdownDogu(name string) *v2.Dogu {
	return &v2.Dogu{Name: name, Namespace: shutdownTestNamespace, Generation: 2}
}

func v3ShutdownDogu(name string, apiVersion v3beta1.DoguApiVersion) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name:       name,
		Namespace:  shutdownTestNamespace,
		Generation: 3,
		Spec:       v3beta1.DoguSpec{Name: name, DoguApiVersion: apiVersion},
		Status: v3beta1.DoguStatus{
			Conditions: []metav1.Condition{{
				Type:    v3beta1.ConditionHealthy,
				Status:  metav1.ConditionTrue,
				Reason:  v3beta1.ReasonSucceeded,
				Message: "all workloads are ready",
			}}},
	}
}

func persistedV3Healthy(t *testing.T, c client.Client, name string) *metav1.Condition {
	t.Helper()
	dogu := &v3beta1.Dogu{}
	require.NoError(t, c.Get(testCtx, client.ObjectKey{Namespace: shutdownTestNamespace, Name: name}, dogu))
	return meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionHealthy)
}

func assertV2Unknown(t *testing.T, c client.Client, name string) {
	t.Helper()
	dogu := &v2.Dogu{}
	require.NoError(t, c.Get(testCtx, client.ObjectKey{Namespace: shutdownTestNamespace, Name: name}, dogu))
	assert.Equal(t, v2.UnknownHealthStatus, dogu.Status.Health)

	var expectedConditions []metav1.Condition
	for _, conditionType := range []string{
		v2.ConditionReady,
		v2.ConditionHealthy,
		v2.ConditionSupportMode,
		v2.ConditionMeetsMinVolumeSize,
		v2.ConditionPauseReconciliation,
	} {
		expectedConditions = append(expectedConditions, metav1.Condition{
			Type: conditionType, Status: metav1.ConditionUnknown, ObservedGeneration: 2,
			Reason: "StoppingOperator", Message: "The operator is shutting down",
		})
	}
	gomega.NewWithT(t).Expect(dogu.Status.Conditions).
		To(conditions.MatchConditions(expectedConditions, conditions.IgnoreLastTransitionTime(true)))
}

func assertV3Unknown(t *testing.T, c client.Client, name string) {
	t.Helper()
	dogu := &v3beta1.Dogu{}
	require.NoError(t, c.Get(testCtx, client.ObjectKey{Namespace: shutdownTestNamespace, Name: name}, dogu))

	var expectedConditions []metav1.Condition
	for _, conditionType := range []string{
		v3beta1.ConditionReady,
		v3beta1.ConditionHealthy,
		v3beta1.ConditionPauseReconciliation,
		v3beta1.ConditionStopped,
		v3beta1.ConditionValid,
		v3beta1.ConditionChartAvailable,
		v3beta1.ConditionUpdatePending,
		v3beta1.ConditionSchemaValidationSkipped,
		v3beta1.ConditionExportModeActive,
	} {
		expectedConditions = append(expectedConditions, metav1.Condition{
			Type: conditionType, Status: metav1.ConditionUnknown, ObservedGeneration: 3,
			Reason: "StoppingOperator", Message: "The operator is shutting down",
		})
	}
	gomega.NewWithT(t).Expect(dogu.Status.Conditions).
		To(conditions.MatchConditions(expectedConditions, conditions.IgnoreLastTransitionTime(true)))
}

func TestNewShutdownHandler(t *testing.T) {
	t.Run("should take namespace and v3 flag from config", func(t *testing.T) {
		handler := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true})

		assert.Equal(t, shutdownTestNamespace, handler.namespace)
		assert.True(t, handler.doguV3Enabled)
	})
}

func TestShutdownHandler_Handle(t *testing.T) {
	t.Run("should set v2 health and conditions to unknown", func(t *testing.T) {
		c := shutdownTestClient(t, interceptor.Funcs{}, v2ShutdownDogu("ldap"), v2ShutdownDogu("cas"))

		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true}).Handle(testCtx, c)

		require.NoError(t, err)
		assertV2Unknown(t, c, "ldap")
	})

	t.Run("should not touch v3 dogus in the v2 flow", func(t *testing.T) {
		v2DoguWithV3Annotation := v2ShutdownDogu("ldap")
		v2DoguWithV3Annotation.Annotations = map[string]string{"k8s.cloudogu.com/v3beta1-doguApiVersion": "v3"}
		c := shutdownTestClient(t, interceptor.Funcs{}, v2DoguWithV3Annotation, v3ShutdownDogu("grafana", v3beta1.DoguApiVersionV3))

		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: false}).Handle(testCtx, c)

		require.NoError(t, err)
		dogu := &v2.Dogu{}
		require.NoError(t, c.Get(testCtx, client.ObjectKey{Namespace: shutdownTestNamespace, Name: "ldap"}, dogu))
		assert.Empty(t, dogu.Status.Conditions)
		doguV3 := &v3beta1.Dogu{}
		require.NoError(t, c.Get(testCtx, client.ObjectKey{Namespace: shutdownTestNamespace, Name: "grafana"}, doguV3))
		assert.Equal(t, metav1.ConditionTrue, persistedV3Healthy(t, c, "grafana").Status)
	})

	t.Run("should set conditions of v3 dogus to unknown and skip v2 dogus in the v3 flow", func(t *testing.T) {
		c := shutdownTestClient(t, interceptor.Funcs{},
			v3ShutdownDogu("grafana", v3beta1.DoguApiVersionV3),
			v3ShutdownDogu("ldap", v3beta1.DoguApiVersionV2),
		)
		// the fake client keeps v2 and v3beta1 objects apart, so the v3 flow is called directly to test its own filter
		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true}).handleV3(testCtx, c)

		require.NoError(t, err)
		assertV3Unknown(t, c, "grafana")
		assert.Equal(t, metav1.ConditionTrue, persistedV3Healthy(t, c, "ldap").Status)
	})

	t.Run("should retry on conflict", func(t *testing.T) {
		conflicts := map[string]int{}
		c := shutdownTestClient(t, interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				// create one conflict for each kind
				kind := "v2"
				if _, ok := obj.(*v3beta1.Dogu); ok {
					kind = "v3"
				}
				conflicts[kind]++
				if conflicts[kind] == 1 {
					return apierrors.NewConflict(schema.GroupResource{Resource: "dogus"}, obj.GetName(), assert.AnError)
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}, v2ShutdownDogu("ldap"), v3ShutdownDogu("grafana", v3beta1.DoguApiVersionV3))

		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true}).Handle(testCtx, c)

		require.NoError(t, err)
		assert.Equal(t, map[string]int{"v2": 2, "v3": 2}, conflicts)
		assertV2Unknown(t, c, "ldap")
		assertV3Unknown(t, c, "grafana")
	})

	t.Run("should continue with other dogus if one update fails", func(t *testing.T) {
		c := shutdownTestClient(t, interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if obj.GetName() == "cas" || obj.GetName() == "grafana" {
					return assert.AnError
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		},
			v2ShutdownDogu("cas"),
			v2ShutdownDogu("ldap"),
			v3ShutdownDogu("grafana", v3beta1.DoguApiVersionV3),
			v3ShutdownDogu("nexus", v3beta1.DoguApiVersionV3),
		)

		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true}).Handle(testCtx, c)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to set health status and conditions of "cas" to unknown`)
		assert.ErrorContains(t, err, `failed to set conditions of v3 dogu "grafana" to unknown`)
		assertV2Unknown(t, c, "ldap")
		assertV3Unknown(t, c, "nexus")
	})

	t.Run("should fail to list dogus", func(t *testing.T) {
		c := shutdownTestClient(t, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		})

		err := NewShutdownHandler(&config.OperatorConfig{Namespace: shutdownTestNamespace, DoguV3Enabled: true}).Handle(testCtx, c)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to list v2 dogus")
		assert.ErrorContains(t, err, "failed to list v3 dogus")
	})
}
