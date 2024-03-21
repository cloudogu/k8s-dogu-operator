package health

import (
	"context"
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewDoguStatusUpdater(t *testing.T) {
	// given
	ecosystemClientMock := mocks.NewEcosystemInterface(t)
	recorderMock := extMocks.NewEventRecorder(t)

	// when
	actual := NewDoguStatusUpdater(ecosystemClientMock, recorderMock)

	// then
	assert.NotEmpty(t, actual)
}

func TestDoguStatusUpdater_UpdateStatus(t *testing.T) {
	t.Run("should fail to get dogu resource", func(t *testing.T) {
		// given
		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(nil, assert.AnError)
		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock}

		// when
		err := sut.UpdateStatus(testCtx, types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}, true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/my-dogu\"")
	})
	t.Run("should fail to update health status of dogu", func(t *testing.T) {
		// given
		dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
		doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1api.UpdateOptions) {
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, v1.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "available", Stopped: false}, status)
			})
		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.EXPECT().Event(dogu, "Warning", "HealthStatusUpdate", "failed to update dogu \"test-namespace/my-dogu\" with current health status [\"\"] to desired health status [\"available\"]")

		sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

		// when
		err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu \"test-namespace/my-dogu\" with current health status [\"\"] to desired health status [\"available\"]")
	})
	t.Run("should succeed to update health status of dogu", func(t *testing.T) {
		t.Run("available", func(t *testing.T) {
			// given
			const expectedHealthStatus = v1.HealthStatus("available")
			dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

			doguClientMock := mocks.NewDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, nil).
				Run(func(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1api.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, v1.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "available", Stopped: false}, status)
					dogu.Status.Health = status.Health
				})
			ecosystemClientMock := mocks.NewEcosystemInterface(t)
			ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

			recorderMock := extMocks.NewEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", expectedHealthStatus)

			sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), true)

			// then
			require.NoError(t, err)
			assert.Equal(t, expectedHealthStatus, dogu.Status.Health)
		})
		t.Run("unavailable", func(t *testing.T) {
			// given
			const expectedHealthStatus = v1.HealthStatus("unavailable")
			dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

			doguClientMock := mocks.NewDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, nil).
				Run(func(ctx context.Context, dogu *v1.Dogu, modifyStatusFn func(v1.DoguStatus) v1.DoguStatus, opts metav1api.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, v1.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "unavailable", Stopped: false}, status)
					dogu.Status.Health = status.Health
				})
			ecosystemClientMock := mocks.NewEcosystemInterface(t)
			ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

			recorderMock := extMocks.NewEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", expectedHealthStatus)

			sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), false)

			// then
			require.NoError(t, err)
			assert.Equal(t, expectedHealthStatus, dogu.Status.Health)
		})
	})
}
