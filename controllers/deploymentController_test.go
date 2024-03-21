package controllers

import (
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
)

func TestNewDeploymentReconciler(t *testing.T) {
	t.Run("should not be empty", func(t *testing.T) {
		// given
		clientSetMock := extMocks.NewClientSet(t)
		availabilityCheckerMock := &health.AvailabilityChecker{}
		healthStatusUpdaterMock := mocks.NewDoguHealthStatusUpdater(t)

		// when
		actual := NewDeploymentReconciler(clientSetMock, availabilityCheckerMock, healthStatusUpdaterMock)

		// then
		assert.NotEmpty(t, actual)
	})
}

func Test_deploymentReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &DeploymentReconciler{}

		// when
		err := sut.SetupWithManager(nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "must provide a non-nil Manager")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		ctrlManMock := extMocks.NewControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createScheme(t))
		logger := log.FromContext(testCtx)
		ctrlManMock.EXPECT().GetLogger().Return(logger)
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)

		sut := &DeploymentReconciler{}

		// when
		err := sut.SetupWithManager(ctrlManMock)

		// then
		require.NoError(t, err)
	})
}

func createScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("apps/v1")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &appsv1.Deployment{})
	return scheme
}

func TestDeploymentReconciler_Reconcile(t *testing.T) {
	t.Run("should fail to get deployment", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		deployClientMock := extMocks.NewDeploymentInterface(t)
		deployClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(nil, assert.AnError)
		appsV1Client := extMocks.NewAppsV1Interface(t)
		appsV1Client.EXPECT().Deployments(testNamespace).Return(deployClientMock)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Client)

		sut := &DeploymentReconciler{
			k8sClientSet: clientSetMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get deployment \"test-namespace/my-dogu\"")
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should ignore deployment not found", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		notFoundErr := errors.NewNotFound(schema.GroupResource{
			Group:    "apps/v1",
			Resource: "Deployment",
		}, "my-dogu")

		deployClientMock := extMocks.NewDeploymentInterface(t)
		deployClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(nil, notFoundErr)
		appsV1Client := extMocks.NewAppsV1Interface(t)
		appsV1Client.EXPECT().Deployments(testNamespace).Return(deployClientMock)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Client)

		sut := &DeploymentReconciler{
			k8sClientSet: clientSetMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should ignore non-dogu deployments", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "not-a-dogu", Namespace: testNamespace}}

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "not-a-dogu",
				Labels: map[string]string{"not-a-dogu-label": "some_value"},
			},
		}

		deployClientMock := extMocks.NewDeploymentInterface(t)
		deployClientMock.EXPECT().Get(testCtx, "not-a-dogu", metav1.GetOptions{}).Return(deployment, nil)
		appsV1Client := extMocks.NewAppsV1Interface(t)
		appsV1Client.EXPECT().Deployments(testNamespace).Return(deployClientMock)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Client)

		sut := &DeploymentReconciler{
			k8sClientSet: clientSetMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail to update dogu health", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-dogu",
				Labels: map[string]string{"dogu.name": "my-dogu", "dogu.version": "1.2.3"},
			},
		}

		deployClientMock := extMocks.NewDeploymentInterface(t)
		deployClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(deployment, nil)
		appsV1Client := extMocks.NewAppsV1Interface(t)
		appsV1Client.EXPECT().Deployments(testNamespace).Return(deployClientMock)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Client)

		deployAvailCheckMock := mocks.NewDeploymentAvailabilityChecker(t)
		deployAvailCheckMock.EXPECT().IsAvailable(deployment).Return(true)

		doguHealthUpdaterMock := mocks.NewDoguHealthStatusUpdater(t)
		doguHealthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Namespace: "", Name: "my-dogu"}, true).Return(assert.AnError)

		sut := &DeploymentReconciler{
			k8sClientSet:            clientSetMock,
			availabilityChecker:     deployAvailCheckMock,
			doguHealthStatusUpdater: doguHealthUpdaterMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu health for deployment \"test-namespace/my-dogu\"")
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should succeed to update dogu health", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "my-dogu",
				Labels: map[string]string{"dogu.name": "my-dogu", "dogu.version": "1.2.3"},
			},
		}

		deployClientMock := extMocks.NewDeploymentInterface(t)
		deployClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(deployment, nil)
		appsV1Client := extMocks.NewAppsV1Interface(t)
		appsV1Client.EXPECT().Deployments(testNamespace).Return(deployClientMock)
		clientSetMock := extMocks.NewClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appsV1Client)

		deployAvailCheckMock := mocks.NewDeploymentAvailabilityChecker(t)
		deployAvailCheckMock.EXPECT().IsAvailable(deployment).Return(false)

		doguHealthUpdaterMock := mocks.NewDoguHealthStatusUpdater(t)
		doguHealthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Namespace: "", Name: "my-dogu"}, false).Return(nil)

		sut := &DeploymentReconciler{
			k8sClientSet:            clientSetMock,
			availabilityChecker:     deployAvailCheckMock,
			doguHealthStatusUpdater: doguHealthUpdaterMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})
}
