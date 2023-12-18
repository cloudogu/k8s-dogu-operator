package health

import (
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewDoguStatusUpdater(t *testing.T) {
	// given
	doguClientMock := mocks.NewDoguInterface(t)
	recorderMock := extMocks.NewEventRecorder(t)

	// when
	actual := NewDoguStatusUpdater(doguClientMock, recorderMock)

	// then
	assert.NotEmpty(t, actual)
}

func TestDoguStatusUpdater_UpdateStatus(t *testing.T) {
	t.Run("should fail to get dogu resource", func(t *testing.T) {
		// given
		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(nil, assert.AnError)

		sut := &DoguStatusUpdater{doguClient: doguClientMock}

		// when
		err := sut.UpdateStatus(testCtx, "my-dogu", true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu resource \"my-dogu\"")
	})
	t.Run("should fail to update health status of dogu", func(t *testing.T) {
		// given
		dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu"}}

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
		doguClientMock.EXPECT().UpdateStatus(testCtx, dogu, metav1api.UpdateOptions{}).Return(nil, assert.AnError)

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.EXPECT().Event(dogu, "Warning", "HealthStatusUpdate", "failed to update dogu \"my-dogu\" with health status \"available\"")

		sut := &DoguStatusUpdater{doguClient: doguClientMock, recorder: recorderMock}

		// when
		err := sut.UpdateStatus(testCtx, "my-dogu", true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu \"my-dogu\" with health status \"available\"")
	})
	t.Run("should succeed to update health status of dogu", func(t *testing.T) {
		t.Run("available", func(t *testing.T) {
			// given
			const expectedHealthStatus = v1.HealthStatus("available")
			dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu"}}

			doguClientMock := mocks.NewDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatus(testCtx, dogu, metav1api.UpdateOptions{}).Return(dogu, nil)

			recorderMock := extMocks.NewEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", expectedHealthStatus)

			sut := &DoguStatusUpdater{doguClient: doguClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, "my-dogu", true)

			// then
			require.NoError(t, err)
			assert.Equal(t, dogu.Status.Health, expectedHealthStatus)
		})
		t.Run("unavailable", func(t *testing.T) {
			// given
			const expectedHealthStatus = v1.HealthStatus("unavailable")
			dogu := &v1.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu"}}

			doguClientMock := mocks.NewDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatus(testCtx, dogu, metav1api.UpdateOptions{}).Return(dogu, nil)

			recorderMock := extMocks.NewEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", expectedHealthStatus)

			sut := &DoguStatusUpdater{doguClient: doguClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, "my-dogu", false)

			// then
			require.NoError(t, err)
			assert.Equal(t, dogu.Status.Health, expectedHealthStatus)
		})
	})
}