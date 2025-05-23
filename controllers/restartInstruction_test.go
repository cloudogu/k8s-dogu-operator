package controllers

import (
	"context"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func Test_restartInstruction_execute(t *testing.T) {
	t.Run("should terminate on operation ignore", func(t *testing.T) {
		// given
		sut := restartInstruction{op: ignore}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should requeue on operation wait", func(t *testing.T) {
		// given
		sut := restartInstruction{op: wait}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueWaitTimeout}, result)
	})

	t.Run("should terminate on get dogu restart error", func(t *testing.T) {
		// given
		sut := restartInstruction{op: handleGetDoguRestartFailed, err: errors.NewNotFound(schema.GroupResource{}, "")}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should terminate on unknown operation", func(t *testing.T) {
		// given
		sut := restartInstruction{op: "unknown"}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should requeue if not already stopped", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: false}}
		sut := restartInstruction{op: checkStopped, dogu: dogu}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueWaitTimeout}, result)
	})

	t.Run("should set restart to stopped if stopped and immediately retry", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: true}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseStopped, status.Phase)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeNormal, "Stopped", "dogu stopped, restarting")

		sut := restartInstruction{op: checkStopped, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true}, result)
	})

	t.Run("should requeue if stopped", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: true}}
		sut := restartInstruction{op: checkStarted, dogu: dogu}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueWaitTimeout}, result)
	})

	t.Run("should set restart to stopped if stopped and immediately retry", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: false}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseCompleted, status.Phase)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeNormal, "Started", "dogu started, restart completed")

		sut := restartInstruction{op: checkStarted, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should stop on stop operation", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: false}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseStopping, status.Phase)
			})

		doguInterface := newMockDoguInterface(t)
		doguInterface.EXPECT().UpdateSpecWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *doguv2.Dogu, modifySpecFn func(doguv2.DoguSpec) doguv2.DoguSpec, opts metav1.UpdateOptions) {
				spec := modifySpecFn(dogu.Spec)
				assert.Equal(t, true, spec.Stopped)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeNormal, "Stopping", "initiated stop of dogu")

		sut := restartInstruction{op: stop, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock, doguInterface: doguInterface}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{RequeueAfter: requeueWaitTimeout}, result)
	})

	t.Run("should set failed status on update stopped spec", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: false}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseFailedStop, status.Phase)
			})

		doguInterface := newMockDoguInterface(t)
		doguInterface.EXPECT().UpdateSpecWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *doguv2.Dogu, modifySpecFn func(doguv2.DoguSpec) doguv2.DoguSpec, opts metav1.UpdateOptions) {
				spec := modifySpecFn(dogu.Spec)
				assert.Equal(t, true, spec.Stopped)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeWarning, "Stopping", "failed to stop dogu")

		sut := restartInstruction{op: stop, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock, doguInterface: doguInterface}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.Error(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should start on start operation", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: true}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseStarting, status.Phase)
			})

		doguInterface := newMockDoguInterface(t)
		doguInterface.EXPECT().UpdateSpecWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *doguv2.Dogu, modifySpecFn func(doguv2.DoguSpec) doguv2.DoguSpec, opts metav1.UpdateOptions) {
				spec := modifySpecFn(dogu.Spec)
				assert.Equal(t, false, spec.Stopped)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeNormal, "Starting", "initiated start of dogu")

		sut := restartInstruction{op: start, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock, doguInterface: doguInterface}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})

	t.Run("should set failed status on update started spec", func(t *testing.T) {
		// given
		dogu := &doguv2.Dogu{Status: doguv2.DoguStatus{Stopped: true}}
		doguRestart := &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{}}

		doguRestartInterface := newMockDoguRestartInterface(t)
		doguRestartInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseFailedStart, status.Phase)
			})

		doguInterface := newMockDoguInterface(t)
		doguInterface.EXPECT().UpdateSpecWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *doguv2.Dogu, modifySpecFn func(doguv2.DoguSpec) doguv2.DoguSpec, opts metav1.UpdateOptions) {
				spec := modifySpecFn(dogu.Spec)
				assert.Equal(t, false, spec.Stopped)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeWarning, "Starting", "failed to start dogu")

		sut := restartInstruction{op: start, dogu: dogu, restart: doguRestart, doguRestartInterface: doguRestartInterface, recorder: eventRecorderMock, doguInterface: doguInterface}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.Error(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})
}

func Test_restartInstruction_handleGetDoguFailed(t *testing.T) {
	t.Run("should update status to failed get dogu", func(t *testing.T) {
		// given
		doguRestart := &doguv2.DoguRestart{}

		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestartInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseFailedGetDogu, status.Phase)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeWarning, "FailedGetDogu", "Could not get ressource of dogu to restart.")

		sut := restartInstruction{op: handleGetDoguFailed, doguRestartInterface: doguRestartInterfaceMock, restart: doguRestart, recorder: eventRecorderMock}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})
}

func Test_restartInstruction_handleDoguNotFound(t *testing.T) {
	t.Run("should update status to dogu not found", func(t *testing.T) {
		// given
		doguRestart := &doguv2.DoguRestart{}

		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestartInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, doguRestart *doguv2.DoguRestart, modifyStatusFn func(doguv2.DoguRestartStatus) doguv2.DoguRestartStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(doguRestart.Status)
				assert.Equal(t, doguv2.RestartStatusPhaseDoguNotFound, status.Phase)
			})

		eventRecorderMock := newMockEventRecorder(t)
		eventRecorderMock.EXPECT().Event(doguRestart, v1.EventTypeWarning, "DoguNotFound", "Dogu to restart was not found.")

		sut := restartInstruction{op: handleDoguNotFound, doguRestartInterface: doguRestartInterfaceMock, restart: doguRestart, recorder: eventRecorderMock}

		// when
		result, err := sut.execute(testCtx)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
	})
}
