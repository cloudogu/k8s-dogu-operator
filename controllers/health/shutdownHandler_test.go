package health

import (
	"context"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShutdownHandler_Start(t *testing.T) {
	t.Run("should update all dogu health status to unknown on shutdown", func(t *testing.T) {
		// given
		doneCtx, cancelFunc := context.WithCancel(testCtx)
		cancelFunc()
		expectedContext := context.WithoutCancel(doneCtx)
		doguInterfaceMock := newMockDoguInterface(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}
		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(expectedContext, metav1.ListOptions{}).Return(doguList, nil)

		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(expectedContext, casDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(casDogu.Status)
				assert.Equal(t, v2.UnknownHealthStatus, status.Health)
			})

		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(expectedContext, ldapDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(ldapDogu.Status)
				assert.Equal(t, v2.UnknownHealthStatus, status.Health)
			})

		sut := ShutdownHandler{doguInterface: doguInterfaceMock}

		// when
		err := sut.Start(doneCtx)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on list error", func(t *testing.T) {
		// given
		doneCtx, cancelFunc := context.WithCancel(testCtx)
		cancelFunc()

		expectedContext := context.WithoutCancel(doneCtx)
		doguInterfaceMock := newMockDoguInterface(t)

		doguInterfaceMock.EXPECT().List(expectedContext, metav1.ListOptions{}).Return(nil, assert.AnError)

		sut := ShutdownHandler{doguInterface: doguInterfaceMock}

		// when
		err := sut.Start(doneCtx)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should join update errors", func(t *testing.T) {
		// given
		doneCtx, cancelFunc := context.WithCancel(testCtx)
		cancelFunc()

		expectedContext := context.WithoutCancel(doneCtx)
		doguInterfaceMock := newMockDoguInterface(t)

		casDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "cas"},
			Status:     v2.DoguStatus{},
		}
		ldapDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap"},
			Status:     v2.DoguStatus{},
		}
		doguList := &v2.DoguList{Items: []v2.Dogu{*casDogu, *ldapDogu}}
		doguInterfaceMock.EXPECT().List(expectedContext, metav1.ListOptions{}).Return(doguList, nil)

		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(expectedContext, casDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(casDogu.Status)
				assert.Equal(t, v2.UnknownHealthStatus, status.Health)
			})

		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(expectedContext, ldapDogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(ldapDogu.Status)
				assert.Equal(t, v2.UnknownHealthStatus, status.Health)
			})

		sut := ShutdownHandler{doguInterface: doguInterfaceMock}

		// when
		err := sut.Start(doneCtx)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set health status of \"cas\" to \"unknown\": assert.AnError general error for testing\nfailed to set health status of \"ldap\" to \"unknown\": assert.AnError general error for testing")
	})
}

func TestNewShutdownHandler(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().Add(mock.Anything).Return(nil)

		// when
		handler, err := NewShutdownHandler(managerMock, doguInterfaceMock)

		// then
		assert.Equal(t, doguInterfaceMock, handler.doguInterface)
		assert.NoError(t, err)
	})
	t.Run("should fail to add handler", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().Add(mock.Anything).Return(assert.AnError)

		// when
		_, err := NewShutdownHandler(managerMock, doguInterfaceMock)

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

}
