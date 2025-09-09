package health

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewStartupHandler(t *testing.T) {
	t.Run("should set properties", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		availabilityCheckerMock := NewMockDeploymentAvailabilityChecker(t)
		healthUpdaterMock := NewMockDoguHealthStatusUpdater(t)

		// when
		handler := NewStartupHandler(doguInterfaceMock, deploymentInterfaceMock, availabilityCheckerMock, healthUpdaterMock)

		// then
		assert.Equal(t, doguInterfaceMock, handler.doguInterface)
	})
}

func TestStartupHandler_Start(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		availabilityCheckerMock := NewMockDeploymentAvailabilityChecker(t)
		healthUpdaterMock := NewMockDoguHealthStatusUpdater(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}

		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)

		casDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "cas"}}
		ldapDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}}
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(casDeploy, nil)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(ldapDeploy, nil)

		availabilityCheckerMock.EXPECT().IsAvailable(casDeploy).Return(true)
		availabilityCheckerMock.EXPECT().IsAvailable(ldapDeploy).Return(false)

		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: "cas"}, true).Return(nil)
		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: "ldap"}, false).Return(nil)

		sut := StartupHandler{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, availabilityChecker: availabilityCheckerMock, doguHealthStatusUpdater: healthUpdaterMock}

		// when
		err := sut.Start(testCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on dogu list error", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		availabilityCheckerMock := NewMockDeploymentAvailabilityChecker(t)
		healthUpdaterMock := NewMockDoguHealthStatusUpdater(t)

		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)

		sut := StartupHandler{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, availabilityChecker: availabilityCheckerMock, doguHealthStatusUpdater: healthUpdaterMock}

		// when
		err := sut.Start(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should return error on deployment get error", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		availabilityCheckerMock := NewMockDeploymentAvailabilityChecker(t)
		healthUpdaterMock := NewMockDoguHealthStatusUpdater(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}

		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)

		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(nil, assert.AnError)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(nil, assert.AnError)

		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: casDogu.Name, Namespace: casDogu.Namespace}, false).Return(nil)
		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: ldapDogu.Name, Namespace: ldapDogu.Namespace}, false).Return(nil)

		sut := StartupHandler{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, availabilityChecker: availabilityCheckerMock, doguHealthStatusUpdater: healthUpdaterMock}

		// when
		err := sut.Start(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment \"cas\": assert.AnError general error for testing\nfailed to get deployment \"ldap\": assert.AnError general error for testing")
	})

	t.Run("should return error on status update error", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		availabilityCheckerMock := NewMockDeploymentAvailabilityChecker(t)
		healthUpdaterMock := NewMockDoguHealthStatusUpdater(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}

		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(doguList, nil)

		casDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "cas"}}
		ldapDeploy := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap"}}
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(casDeploy, nil)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "ldap", metav1.GetOptions{}).Return(ldapDeploy, nil)

		availabilityCheckerMock.EXPECT().IsAvailable(casDeploy).Return(true)
		availabilityCheckerMock.EXPECT().IsAvailable(ldapDeploy).Return(false)

		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: "cas"}, true).Return(assert.AnError)
		healthUpdaterMock.EXPECT().UpdateStatus(testCtx, types.NamespacedName{Name: "ldap"}, false).Return(assert.AnError)

		sut := StartupHandler{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, availabilityChecker: availabilityCheckerMock, doguHealthStatusUpdater: healthUpdaterMock}

		// when
		err := sut.Start(testCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to refresh health status of \"cas\": assert.AnError general error for testing\nfailed to refresh health status of \"ldap\": assert.AnError general error for testing")
	})
}
