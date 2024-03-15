package controllers

// func TestDoguRestartReconciler_handleDoguNotFound(t *testing.T) {
// 	t.Run("success", func(t *testing.T) {
// 		// given
// 		eventRecorderMock := extmocks.NewEventRecorder(t)
// 		doguRestartInterfaceMock := mocks.NewDoguRestartInterface(t)
//
// 		sut := DoguRestartReconciler{recorder: eventRecorderMock, doguRestartInterface: doguRestartInterfaceMock}
//
// 		doguRestart := &v1.DoguRestart{}
// 		instruction := restartInstruction{restart: doguRestart}
//
// 		eventRecorderMock.EXPECT().Event(doguRestart, "Warning", "DoguNotFound", "Dogu to restart was not found.")
// 		doguRestartInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, doguRestart, func(status v1.DoguRestartStatus) v1.DoguRestartStatus {
// 			status.Phase = v1.RestartStatusPhaseDoguNotFound
// 			return status
// 		}, metav1.UpdateOptions{}).Return(nil, nil)
//
// 		// when
// 		result, err := sut.handleDoguNotFound(testCtx, instruction)
//
// 		// then
// 		require.NoError(t, err)
// 		assert.Equal(t, ctrl.Result{}, result)
// 	})
// }
