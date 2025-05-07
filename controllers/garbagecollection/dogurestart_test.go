package garbagecollection

import (
	"context"
	"fmt"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"testing"
	"time"
)

var testCtx = context.Background()

func TestDoguRestartGarbageCollector_DoGarbageCollection(t *testing.T) {
	t.Run("should return on invalid gc disabled flag", func(t *testing.T) {
		// given
		err := os.Setenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED", "2")
		require.NoError(t, err)
		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		sut := DoguRestartGarbageCollector{}

		// when
		err = sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to convert garbage collection disabled flag \"2\" of dogu restarts")
	})

	t.Run("should do nothing if gc is disabled", func(t *testing.T) {
		// given
		err := os.Setenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED", "true")
		require.NoError(t, err)
		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		sut := DoguRestartGarbageCollector{}

		// when
		err = sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("should execute gc with disabled flag false", func(t *testing.T) {
		// given
		err := os.Setenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED", "false")
		require.NoError(t, err)
		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_GARBAGE_COLLECTION_DISABLED")
			require.NoError(t, unsetErr)
		}()

		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		list := &v2.DoguRestartList{Items: []v2.DoguRestart{}}
		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(list, nil)

		doguName := "ldap"
		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err = sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("should return an error on invalid limit", func(t *testing.T) {
		// given
		envErr := os.Setenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT", "-2")
		require.NoError(t, envErr)

		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		now := metav1.Now()
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)
		list := &v2.DoguRestartList{Items: []v2.DoguRestart{getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseCompleted, now.Add(time.Second))}}
		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(list, nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to execute garbage collection because the limit is less than 0: -2")
	})

	t.Run("Should keep last 3 successful resources for dogu (default)", func(t *testing.T) {
		// given
		doguName := "ldap"
		otherDoguName := "cas"
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)

		now := metav1.Now()

		restartList := &v2.DoguRestartList{
			Items: []v2.DoguRestart{
				getDoguRestartWithCreationTimestamp(doguName, "3", v2.RestartStatusPhaseCompleted, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "2", v2.RestartStatusPhaseCompleted, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseCompleted, now.Add(time.Second)),
				getDoguRestartWithCreationTimestamp(doguName, "5", v2.RestartStatusPhaseCompleted, now.Add(time.Second*5)),
				getDoguRestartWithCreationTimestamp(doguName, "4", v2.RestartStatusPhaseCompleted, now.Add(time.Second*4)),
				getDoguRestartWithCreationTimestamp(otherDoguName, "1", v2.RestartStatusPhaseCompleted, now.Add(time.Second)),
			},
		}

		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(restartList, nil).Times(1)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-1", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-2", doguName), metav1.DeleteOptions{}).Return(nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("Should keep last n successful and failed resources for different dogus", func(t *testing.T) {
		// given
		envErr := os.Setenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT", "1")
		require.NoError(t, envErr)

		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT")
			require.NoError(t, unsetErr)
		}()

		envErr = os.Setenv("DOGU_RESTART_FAILED_HISTORY_LIMIT", "1")
		require.NoError(t, envErr)

		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_FAILED_HISTORY_LIMIT")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)

		now := metav1.Now()

		restartList := &v2.DoguRestartList{
			Items: []v2.DoguRestart{
				getDoguRestartWithCreationTimestamp(doguName, "3", v2.RestartStatusPhaseCompleted, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "2", v2.RestartStatusPhaseCompleted, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseCompleted, now.Add(time.Second)),
				getDoguRestartWithCreationTimestamp(doguName, "6", v2.RestartStatusPhaseFailedGetDogu, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "5", v2.RestartStatusPhaseFailedStart, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "4", v2.RestartStatusPhaseFailedStop, now.Add(time.Second)),
			},
		}

		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(restartList, nil).Times(1)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-1", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-2", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-4", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-5", doguName), metav1.DeleteOptions{}).Return(nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("Should keep last 3 failed resources for dogu (default)", func(t *testing.T) {
		// given
		doguName := "ldap"
		otherDoguName := "cas"
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)

		now := metav1.Now()

		restartList := &v2.DoguRestartList{
			Items: []v2.DoguRestart{
				getDoguRestartWithCreationTimestamp(doguName, "3", v2.RestartStatusPhaseDoguNotFound, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "2", v2.RestartStatusPhaseFailedGetDogu, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseFailedStart, now.Add(time.Second)),
				getDoguRestartWithCreationTimestamp(doguName, "5", v2.RestartStatusPhaseFailedStop, now.Add(time.Second*5)),
				getDoguRestartWithCreationTimestamp(doguName, "4", v2.RestartStatusPhaseFailedStop, now.Add(time.Second*4)),
				getDoguRestartWithCreationTimestamp(otherDoguName, "1", v2.RestartStatusPhaseFailedStop, now.Add(time.Second)),
			},
		}

		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(restartList, nil).Times(1)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-1", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-2", doguName), metav1.DeleteOptions{}).Return(nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("Should keep last n successful resources for dogu", func(t *testing.T) {
		// given
		envErr := os.Setenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT", "2")
		require.NoError(t, envErr)

		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_SUCCESSFUL_HISTORY_LIMIT")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)

		now := metav1.Now()

		restartList := &v2.DoguRestartList{
			Items: []v2.DoguRestart{
				getDoguRestartWithCreationTimestamp(doguName, "3", v2.RestartStatusPhaseCompleted, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "2", v2.RestartStatusPhaseCompleted, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseCompleted, now.Add(time.Second)),
				getDoguRestartWithCreationTimestamp(doguName, "5", v2.RestartStatusPhaseCompleted, now.Add(time.Second*5)),
				getDoguRestartWithCreationTimestamp(doguName, "4", v2.RestartStatusPhaseCompleted, now.Add(time.Second*4)),
			},
		}

		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(restartList, nil).Times(1)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-1", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-2", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-3", doguName), metav1.DeleteOptions{}).Return(nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})

	t.Run("Should keep last n failed resources for dogu", func(t *testing.T) {
		// given
		envErr := os.Setenv("DOGU_RESTART_FAILED_HISTORY_LIMIT", "2")
		require.NoError(t, envErr)

		defer func() {
			unsetErr := os.Unsetenv("DOGU_RESTART_FAILED_HISTORY_LIMIT")
			require.NoError(t, unsetErr)
		}()

		doguName := "ldap"
		doguRestartInterfaceMock := newMockDoguRestartInterface(t)

		now := metav1.Now()

		restartList := &v2.DoguRestartList{
			Items: []v2.DoguRestart{
				getDoguRestartWithCreationTimestamp(doguName, "3", v2.RestartStatusPhaseDoguNotFound, now.Add(time.Second*3)),
				getDoguRestartWithCreationTimestamp(doguName, "2", v2.RestartStatusPhaseFailedGetDogu, now.Add(time.Second*2)),
				getDoguRestartWithCreationTimestamp(doguName, "1", v2.RestartStatusPhaseFailedStart, now.Add(time.Second)),
				getDoguRestartWithCreationTimestamp(doguName, "5", v2.RestartStatusPhaseFailedStop, now.Add(time.Second*5)),
				getDoguRestartWithCreationTimestamp(doguName, "4", v2.RestartStatusPhaseFailedStop, now.Add(time.Second*4)),
			},
		}

		doguRestartInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(restartList, nil).Times(1)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-1", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-2", doguName), metav1.DeleteOptions{}).Return(nil)
		doguRestartInterfaceMock.EXPECT().Delete(testCtx, fmt.Sprintf("%s-3", doguName), metav1.DeleteOptions{}).Return(nil)

		sut := DoguRestartGarbageCollector{doguRestartInterface: doguRestartInterfaceMock}

		// when
		err := sut.DoGarbageCollection(testCtx, doguName)

		// then
		require.NoError(t, err)
	})
}

func getDoguRestartWithCreationTimestamp(doguName string, resourceSuffix string, phase v2.RestartStatusPhase, time time.Time) v2.DoguRestart {
	return v2.DoguRestart{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("%s-%s", doguName, resourceSuffix),
			CreationTimestamp: metav1.NewTime(time),
		},
		Spec: v2.DoguRestartSpec{
			DoguName: doguName,
		},
		Status: v2.DoguRestartStatus{
			Phase: phase,
		},
	}
}
