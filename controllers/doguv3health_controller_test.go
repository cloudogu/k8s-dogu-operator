package controllers

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/health"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	healthTestRequest = reconcile.Request{Namespace: testNamespace, Name: testDoguName}
	healthyState      = health.State{Healthy: true, Reason: v3beta1.ReasonSucceeded, Message: "all workloads are ready"}
	unhealthyState    = health.State{Healthy: false, Reason: v3beta1.ReasonWorkloadsNotReady, Message: "workloads not ready"}
)

func healthTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v3beta1.AddToScheme(scheme)
	return scheme
}

func healthTestDogu(apiVersion v3beta1.DoguApiVersion, conditions ...metav1.Condition) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Name: testDoguName, Namespace: testNamespace, Generation: 3,
		Spec:   v3beta1.DoguSpec{Name: testDoguName, DoguApiVersion: apiVersion},
		Status: v3beta1.DoguStatus{Conditions: conditions},
	}
}

func healthyCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               v3beta1.ConditionHealthy,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: 3,
	}
}

func newFakeClient(funcs interceptor.Funcs, objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(healthTestScheme()).
		WithObjects(objects...).
		WithStatusSubresource(&v3beta1.Dogu{}).
		WithInterceptorFuncs(funcs).
		Build()
}

func persistedHealthy(t *testing.T, c client.Client) *metav1.Condition {
	t.Helper()
	persisted := &v3beta1.Dogu{}
	require.NoError(t, c.Get(context.Background(), healthTestRequest.NamespacedName, persisted))
	return meta.FindStatusCondition(persisted.Status.Conditions, v3beta1.ConditionHealthy)
}

func TestNewDoguHealthReconciler(t *testing.T) {
	t.Run("should register the controller with the manager if dogu v3 is enabled", func(t *testing.T) {
		scheme := healthTestScheme()
		require.NoError(t, appsv1.AddToScheme(scheme))
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(scheme)
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(nil)
		managerMock.EXPECT().GetCache().Return(nil)
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		sut, err := NewDoguHealthReconciler(nil, nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should not register the controller with the manager if dogu v3 is disabled", func(t *testing.T) {
		// the manager mock fails the test on any call
		sut, err := NewDoguHealthReconciler(nil, nil, newMockCtrlManager(t), &opConfig.OperatorConfig{DoguV3Enabled: false})

		require.NoError(t, err)
		assert.NotNil(t, sut)
	})

	t.Run("should return error if the controller cannot be registered", func(t *testing.T) {
		scheme := healthTestScheme()
		require.NoError(t, appsv1.AddToScheme(scheme))
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetClient().Return(nil)
		// controller names are validated process-wide, so registering in several tests would fail
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{SkipNameValidation: ptr.To(true)})
		managerMock.EXPECT().GetScheme().Return(scheme)
		managerMock.EXPECT().GetLogger().Return(logr.Logger{})
		managerMock.EXPECT().Add(mock.Anything).Return(assert.AnError)
		managerMock.EXPECT().GetCache().Return(nil).Maybe()
		managerMock.EXPECT().GetRESTMapper().Return(nil).Maybe()

		_, err := NewDoguHealthReconciler(nil, nil, managerMock, &opConfig.OperatorConfig{DoguV3Enabled: true})

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to setup dogu-v3-health controller")
	})
}

func TestDoguHealthReconciler_Reconcile(t *testing.T) {
	t.Run("should do nothing if the dogu does not exist", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{})
		sut := newDoguHealthReconciler(c, newMockHealthChecker(t), newMockEventRecorder(t))

		result, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
	})

	t.Run("should return error if the dogu cannot be fetched", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{
			Get: func(ctx context.Context, _ client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			},
		})
		sut := newDoguHealthReconciler(c, newMockHealthChecker(t), newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu")
	})

	t.Run("should skip v2 dogus", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{}, healthTestDogu(v3beta1.DoguApiVersionV2))
		sut := newDoguHealthReconciler(c, newMockHealthChecker(t), newMockEventRecorder(t))

		result, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		assert.Nil(t, persistedHealthy(t, c))
	})

	t.Run("should return error if the health check fails", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{}, healthTestDogu(v3beta1.DoguApiVersionV3))
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(health.State{}, assert.AnError)
		sut := newDoguHealthReconciler(c, checker, newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check health")
		assert.Nil(t, persistedHealthy(t, c))
	})

	t.Run("should set Healthy to true and emit a normal event on first check", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{}, healthTestDogu(v3beta1.DoguApiVersionV3))
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(healthyState, nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, v3beta1.ReasonSucceeded, "all workloads are ready").Return()
		sut := newDoguHealthReconciler(c, checker, recorder)

		result, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		assert.Equal(t, reconcile.Result{}, result)
		healthy := persistedHealthy(t, c)
		require.NotNil(t, healthy)
		assert.Equal(t, metav1.ConditionTrue, healthy.Status)
		assert.Equal(t, v3beta1.ReasonSucceeded, healthy.Reason)
		assert.Equal(t, "all workloads are ready", healthy.Message)
		assert.Equal(t, int64(3), healthy.ObservedGeneration)
	})

	t.Run("should set Healthy to false but emit no event on first check", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{}, healthTestDogu(v3beta1.DoguApiVersionV3))
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(unhealthyState, nil)
		sut := newDoguHealthReconciler(c, checker, newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		healthy := persistedHealthy(t, c)
		require.NotNil(t, healthy)
		assert.Equal(t, metav1.ConditionFalse, healthy.Status)
		assert.Equal(t, v3beta1.ReasonWorkloadsNotReady, healthy.Reason)
	})

	t.Run("should set Healthy to false and emit a warning event on transition from true", func(t *testing.T) {
		dogu := healthTestDogu(v3beta1.DoguApiVersionV3, healthyCondition(metav1.ConditionTrue, v3beta1.ReasonSucceeded, "all workloads are ready"))
		c := newFakeClient(interceptor.Funcs{}, dogu)
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(unhealthyState, nil)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(mock.Anything, corev1.EventTypeWarning, v3beta1.ReasonWorkloadsNotReady, "workloads not ready").Return()
		sut := newDoguHealthReconciler(c, checker, recorder)

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		healthy := persistedHealthy(t, c)
		require.NotNil(t, healthy)
		assert.Equal(t, metav1.ConditionFalse, healthy.Status)
		assert.Equal(t, v3beta1.ReasonWorkloadsNotReady, healthy.Reason)
	})

	t.Run("should update the message but emit no event if the status does not change", func(t *testing.T) {
		dogu := healthTestDogu(v3beta1.DoguApiVersionV3, healthyCondition(metav1.ConditionFalse, v3beta1.ReasonWorkloadsNotReady, "old message"))
		c := newFakeClient(interceptor.Funcs{}, dogu)
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(unhealthyState, nil)
		sut := newDoguHealthReconciler(c, checker, newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		healthy := persistedHealthy(t, c)
		require.NotNil(t, healthy)
		assert.Equal(t, "workloads not ready", healthy.Message)
	})

	t.Run("should not update the status if the condition is unchanged", func(t *testing.T) {
		dogu := healthTestDogu(v3beta1.DoguApiVersionV3, healthyCondition(metav1.ConditionTrue, v3beta1.ReasonSucceeded, "all workloads are ready"))
		c := newFakeClient(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, _ client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				t.Fatal("unexpected status update")
				return nil
			},
		}, dogu)
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(healthyState, nil)
		sut := newDoguHealthReconciler(c, checker, newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
	})

	t.Run("should retry with a freshly fetched dogu on conflict", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{}, healthTestDogu(v3beta1.DoguApiVersionV3))
		conflicted := false
		c = interceptor.NewClient(c.(client.WithWatch), interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, inner client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if !conflicted {
					conflicted = true
					return apierrors.NewConflict(schema.GroupResource{Resource: "dogus"}, testDoguName, assert.AnError)
				}
				return inner.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		})
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(healthyState, nil).Times(2)
		recorder := newMockEventRecorder(t)
		recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, v3beta1.ReasonSucceeded, "all workloads are ready").Return().Once()
		sut := newDoguHealthReconciler(c, checker, recorder)

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.NoError(t, err)
		assert.True(t, conflicted)
		healthy := persistedHealthy(t, c)
		require.NotNil(t, healthy)
		assert.Equal(t, metav1.ConditionTrue, healthy.Status)
	})

	t.Run("should return error if the status update fails", func(t *testing.T) {
		c := newFakeClient(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, _ client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				return assert.AnError
			},
		}, healthTestDogu(v3beta1.DoguApiVersionV3))
		checker := newMockHealthChecker(t)
		checker.EXPECT().Check(mock.Anything, mock.Anything).Return(healthyState, nil)
		sut := newDoguHealthReconciler(c, checker, newMockEventRecorder(t))

		_, err := sut.Reconcile(context.Background(), healthTestRequest)

		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update Healthy condition")
	})
}

func mappingTestDogu(name, namespace, specName string, apiVersion v3beta1.DoguApiVersion) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		ObjectMeta: objectMeta(name, namespace, ""),
		Spec:       v3beta1.DoguSpec{Name: specName, DoguApiVersion: apiVersion},
	}
}

func newMappingClient(t *testing.T, funcs interceptor.Funcs, dogus ...client.Object) client.Client {
	t.Helper()
	scheme := healthTestScheme()
	require.NoError(t, v3beta1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogus...).WithInterceptorFuncs(funcs).Build()
}

func request(name string) reconcile.Request {
	return reconcile.Request{Namespace: testNamespace, Name: name}
}

func objectMeta(name, namespace, doguName string) metav1.ObjectMeta {
	objMeta := metav1.ObjectMeta{Name: name, Namespace: namespace}
	if doguName != "" {
		objMeta.Labels = map[string]string{v3beta1.DoguLabelName: doguName}
	}
	return objMeta
}

func TestNewDoguRequestMapper(t *testing.T) {
	workload := &appsv1.Deployment{ObjectMeta: objectMeta("nexus-web", testNamespace, testDoguName)}

	t.Run("should map the workload to the v3 dogu with the matching spec name", func(t *testing.T) {
		// the resource name deliberately differs from the spec name, the mapping must use the spec name
		c := newMappingClient(t, interceptor.Funcs{},
			mappingTestDogu("my-nexus", testNamespace, testDoguName, v3beta1.DoguApiVersionV3),
		)

		requests := newDoguRequestMapper(c)(context.Background(), workload)

		assert.Equal(t, []reconcile.Request{request("my-nexus")}, requests)
	})

	t.Run("should map only the matching dogu if there are multiple dogus", func(t *testing.T) {
		c := newMappingClient(t, interceptor.Funcs{},
			mappingTestDogu("ldap", testNamespace, "ldap", v3beta1.DoguApiVersionV3),
			mappingTestDogu(testDoguName, testNamespace, testDoguName, v3beta1.DoguApiVersionV3),
			mappingTestDogu("redmine", testNamespace, "redmine", v3beta1.DoguApiVersionV3),
			mappingTestDogu(testDoguName, "other-namespace", testDoguName, v3beta1.DoguApiVersionV3),
		)

		requests := newDoguRequestMapper(c)(context.Background(), workload)

		assert.Equal(t, []reconcile.Request{request(testDoguName)}, requests)
	})

	t.Run("should map nothing if no dogu matches the label", func(t *testing.T) {
		c := newMappingClient(t, interceptor.Funcs{},
			mappingTestDogu("ldap", testNamespace, "ldap", v3beta1.DoguApiVersionV3),
		)

		requests := newDoguRequestMapper(c)(context.Background(), workload)

		assert.Empty(t, requests)
	})

	t.Run("should map nothing if only a v2 dogu matches the label", func(t *testing.T) {
		c := newMappingClient(t, interceptor.Funcs{},
			mappingTestDogu(testDoguName, testNamespace, testDoguName, v3beta1.DoguApiVersionV2),
		)

		requests := newDoguRequestMapper(c)(context.Background(), workload)

		assert.Empty(t, requests)
	})

	t.Run("should map nothing and not list dogus if the workload has no dogu label", func(t *testing.T) {
		c := newMappingClient(t, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				t.Fatal("dogus must not be listed for a workload without dogu label")
				return nil
			},
		})
		unlabeled := &appsv1.StatefulSet{ObjectMeta: objectMeta("postgres", testNamespace, "")}

		requests := newDoguRequestMapper(c)(context.Background(), unlabeled)

		assert.Empty(t, requests)
	})

	t.Run("should map nothing if the dogus cannot be listed", func(t *testing.T) {
		c := newMappingClient(t, interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		})

		requests := newDoguRequestMapper(c)(context.Background(), workload)

		assert.Empty(t, requests)
	})
}
