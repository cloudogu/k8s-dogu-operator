package controllers

import (
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

const (
	testCasRestartName = "cas-1234"
	testCasDoguName    = "cas"
)

var testCasRestartRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: testCasRestartName}}

func TestDoguRestartReconciler_createRestartInstruction(t *testing.T) {
	t.Run("should return error on error getting restart resource", func(t *testing.T) {
		// given
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
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
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
		doguRestart := &v1.DoguRestart{Status: v1.DoguRestartStatus{Phase: v1.RestartStatusPhaseDoguNotFound}}
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
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
		doguRestart := &v1.DoguRestart{Status: v1.DoguRestartStatus{Phase: v1.RestartStatusPhaseCompleted}}
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
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
		doguRestart := &v1.DoguRestart{Status: v1.DoguRestartStatus{Phase: "unknown"}}
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
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
		doguRestart := &v1.DoguRestart{Spec: v1.DoguRestartSpec{DoguName: testCasDoguName}, Status: v1.DoguRestartStatus{Phase: v1.RestartStatusPhaseStopping}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		doguInterfaceMock := mocks.NewDoguInterface(t)
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
		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
		doguRestart := &v1.DoguRestart{Spec: v1.DoguRestartSpec{DoguName: testCasDoguName}, Status: v1.DoguRestartStatus{Phase: v1.RestartStatusPhaseStopping}}
		doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

		doguInterfaceMock := mocks.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, testCasRestartName))

		sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock, doguInterface: doguInterfaceMock}

		// when
		instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

		// then
		require.Error(t, instruction.err)
		assert.Equal(t, handleDoguNotFound, instruction.op)
	})

	t.Run("should check if stopped on status phase stopping", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseStopping, checkStopped)
	})

	t.Run("should check if started on status phase starting", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseStarting, checkStarted)
	})

	t.Run("should start on status phase stopped", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseStopped, start)
	})

	t.Run("should start on failed start", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseFailedStart, start)
	})

	t.Run("should stop on initial status", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseNew, stop)
	})

	t.Run("should stop on stop failure", func(t *testing.T) {
		testRestartPhaseMapping(t, v1.RestartStatusPhaseFailedStop, stop)
	})
}

func testRestartPhaseMapping(t *testing.T, phase v1.RestartStatusPhase, expectedOperation RestartOperation) {
	// given
	doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
	doguRestart := &v1.DoguRestart{Spec: v1.DoguRestartSpec{DoguName: testCasDoguName}, Status: v1.DoguRestartStatus{Phase: phase}}
	doguRestartInterfaceMock.EXPECT().Get(testCtx, testCasRestartName, metav1.GetOptions{}).Return(doguRestart, nil)

	doguInterfaceMock := mocks.NewDoguInterface(t)
	dogu := &v1.Dogu{}
	doguInterfaceMock.EXPECT().Get(testCtx, testCasDoguName, metav1.GetOptions{}).Return(dogu, nil)

	sut := DoguRestartReconciler{doguRestartInterface: doguRestartInterfaceMock, doguInterface: doguInterfaceMock}

	// when
	instruction := sut.createRestartInstruction(testCtx, testCasRestartRequest)

	// then
	require.NoError(t, instruction.err)
	assert.Equal(t, expectedOperation, instruction.op)
	assert.Equal(t, dogu, instruction.dogu)
}
