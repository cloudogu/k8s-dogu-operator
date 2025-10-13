package controllers

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testCasRestartName = "cas-1234"
	testCasDoguName    = "cas"
)

var testCtx = context.Background()

var testCasRestartRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: testCasRestartName}}

func TestDoguRestartReconciler_createRestartInstruction(t *testing.T) {
	t.Run("should return error on error getting restart resource", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.Error(t, instruction.err)
		assert.Equal(t, handleGetDoguRestartFailed, instruction.op)
	})

	t.Run("should ignore if the dogu restart is not found", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestart := &v2.DoguRestart{Status: v2.DoguRestartStatus{Phase: v2.RestartStatusPhaseDoguNotFound}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.NoError(t, instruction.err)
		assert.Equal(t, ignore, instruction.op)
	})

	t.Run("should ignore if the dogu restart is completed", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestart := &v2.DoguRestart{Status: v2.DoguRestartStatus{Phase: v2.RestartStatusPhaseCompleted}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.NoError(t, instruction.err)
		assert.Equal(t, ignore, instruction.op)
	})

	t.Run("should ignore if the dogu restart has a unknown status phase", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestart := &v2.DoguRestart{Status: v2.DoguRestartStatus{Phase: "unknown"}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.NoError(t, instruction.err)
		assert.Equal(t, ignore, instruction.op)
	})

	t.Run("should return error on get dogu error", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestart := &v2.DoguRestart{Spec: v2.DoguRestartSpec{DoguName: testCasDoguName}, Status: v2.DoguRestartStatus{Phase: v2.RestartStatusPhaseStopping}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		doguInterfaceMock := newMockDoguInterface(t)
		doguInterfaceMock.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(nil, assert.AnError)

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock, doguInterface: doguInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.Error(t, instruction.err)
		assert.Equal(t, handleGetDoguFailed, instruction.op)
	})

	t.Run("should return error on dogu not found", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		doguRestart := &v2.DoguRestart{Spec: v2.DoguRestartSpec{DoguName: testCasDoguName}, Status: v2.DoguRestartStatus{Phase: v2.RestartStatusPhaseStopping}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		doguInterfaceMock := newMockDoguInterface(t)
		doguInterfaceMock.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, testCasRestartName))

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock, doguInterface: doguInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.Error(t, instruction.err)
		assert.Equal(t, handleDoguNotFound, instruction.op)
	})

	t.Run("should check if stopped on status phase stopping", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseStopping, checkStopped)
	})

	t.Run("should check if started on status phase starting", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseStarting, checkStarted)
	})

	t.Run("should start on status phase stopped", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseStopped, start)
	})

	t.Run("should start on failed start", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseFailedStart, start)
	})

	t.Run("should stop on initial status", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseNew, stop)
	})

	t.Run("should stop on stop failure", func(t *testing.T) {
		testRestartPhaseMapping(t, v2.RestartStatusPhaseFailedStop, stop)
	})
}

func testRestartPhaseMapping(t *testing.T, phase v2.RestartStatusPhase, expectedOperation RestartOperation) {
	// given
	doguRestartInterfaceMock := newMockDoguRestartInterface(t)
	doguRestart := &v2.DoguRestart{Spec: v2.DoguRestartSpec{DoguName: testCasDoguName}, Status: v2.DoguRestartStatus{Phase: phase}}
	doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

	doguInterfaceMock := newMockDoguInterface(t)
	dogu := &v2.Dogu{}
	doguInterfaceMock.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(dogu, nil)

	sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock, doguInterface: doguInterfaceMock}

	// when
	instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

	// then
	require.NoError(t, instruction.err)
	assert.Equal(t, expectedOperation, instruction.op)
	assert.Equal(t, dogu, instruction.dogu)
}

func TestNewDoguRestartReconciler(t *testing.T) {
	t.Run("should fail create dogu restart reconciler", func(t *testing.T) {
		// given
		managerMock := newMockCtrlManager(t)
		managerMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		managerMock.EXPECT().GetScheme().Return(getTestScheme())

		// when
		r, err := NewDoguRestartReconciler(
			newMockDoguRestartInterface(t),
			newMockDoguInterface(t),
			newMockEventRecorder(t),
			NewMockDoguRestartGarbageCollector(t),
			managerMock,
		)

		// then
		assert.Empty(t, r)
		assert.Error(t, err)
	})
}

func TestDoguRestartReconciler_Reconcile(t *testing.T) {
	type fields struct {
		doguInterfaceFn        func(t *testing.T) doguInterface
		doguRestartInterfaceFn func(t *testing.T) doguRestartInterface
		garbageCollectorFn     func(t *testing.T) DoguRestartGarbageCollector
		recorderFn             func(t *testing.T) eventRecorder
	}
	tests := []struct {
		name    string
		fields  fields
		req     controllerruntime.Request
		want    controllerruntime.Result
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogu := &v2.Dogu{}
					mck.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(dogu, nil)
					mck.EXPECT().UpdateSpecWithRetry(testCtx, mock.Anything, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguRestartInterfaceFn: func(t *testing.T) doguRestartInterface {
					mck := newMockDoguRestartInterface(t)
					doguRestart := &v2.DoguRestart{Spec: v2.DoguRestartSpec{DoguName: testCasDoguName}, Status: v2.DoguRestartStatus{Phase: v2.RestartStatusPhaseStopped}}
					mck.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				garbageCollectorFn: func(t *testing.T) DoguRestartGarbageCollector {
					return NewMockDoguRestartGarbageCollector(t)
				},
				recorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.Anything, v1.EventTypeWarning, "Starting", "failed to start dogu")
					return mck
				},
			},
			req:     reconcile.Request{NamespacedName: types.NamespacedName{Name: testCasRestartName}},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DoguRestartReconciler{
				doguInterface:        tt.fields.doguInterfaceFn(t),
				doguRestartInterface: tt.fields.doguRestartInterfaceFn(t),
				garbageCollector:     tt.fields.garbageCollectorFn(t),
				recorder:             tt.fields.recorderFn(t),
			}
			got, err := r.Reconcile(testCtx, tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v, %v)", testCtx, tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Reconcile(%v, %v)", testCtx, tt.req)
		})
	}
}
