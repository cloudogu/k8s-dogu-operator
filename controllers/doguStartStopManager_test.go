package controllers

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_doguStartStopManager_StartStopDogu(t *testing.T) {
	testCtx := context.Background()

	t.Run("should stop dogu", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: false,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		dogu := &core.Dogu{Name: "myDogu"}

		mResourceUpserter := newMockResourceUpserter(t)
		mResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, nil)

		mDoguFetcher := newMockLocalDoguFetcher(t)
		mDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "stopping", updatedStatus.Status)
			assert.False(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, doguNotYetStartedStoppedError{doguName: "myDogu", stopped: true})
	})

	t.Run("should start dogu", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: false,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: true,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 0,
			},
		}

		dogu := &core.Dogu{Name: "myDogu"}

		mResourceUpserter := newMockResourceUpserter(t)
		mResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, nil)

		mDoguFetcher := newMockLocalDoguFetcher(t)
		mDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "starting", updatedStatus.Status)
			assert.True(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, doguNotYetStartedStoppedError{doguName: "myDogu", stopped: false})
	})

	t.Run("should update status when dogu is stopped", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "stopping",
				Stopped: true,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 0,
			},
		}

		mResourceUpserter := newMockResourceUpserter(t)
		mDoguFetcher := newMockLocalDoguFetcher(t)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "installed", updatedStatus.Status)
			assert.True(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should update status when dogu is started", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: false,
			},
			Status: doguv2.DoguStatus{
				Status:  "starting",
				Stopped: false,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		mResourceUpserter := newMockResourceUpserter(t)
		mDoguFetcher := newMockLocalDoguFetcher(t)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "installed", updatedStatus.Status)
			assert.False(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should fail to stop dogu i cant get deployment", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: false,
			},
		}

		dogu := &core.Dogu{Name: "myDogu"}

		mResourceUpserter := newMockResourceUpserter(t)
		mResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, nil)

		mDoguFetcher := newMockLocalDoguFetcher(t)
		mDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "stopping", updatedStatus.Status)
			assert.False(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(nil, assert.AnError)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err.(doguNotYetStartedStoppedError).err, assert.AnError)
		assert.ErrorContains(t, err, "error while starting/stopping dogu \"myDogu\": failed to get deployment \"myDogu\":")
	})

	t.Run("should fail to stop dogu for error while updating status", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: false,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		mResourceUpserter := newMockResourceUpserter(t)
		mDoguFetcher := newMockLocalDoguFetcher(t)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update status of dogu \"myDogu\" to \"stopping\":")
	})

	t.Run("should fail to stop dogu for while fetching dogu descriptor", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: false,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		mResourceUpserter := newMockResourceUpserter(t)

		mDoguFetcher := newMockLocalDoguFetcher(t)
		mDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(nil, assert.AnError)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "stopping", updatedStatus.Status)
			assert.False(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get local descriptor for dogu \"myDogu\":")
	})

	t.Run("should fail to stop dogu for while upserting deployment", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec: doguv2.DoguSpec{
				Name:    "official/myDogu",
				Stopped: true,
			},
			Status: doguv2.DoguStatus{
				Status:  "installed",
				Stopped: false,
			},
		}

		deployment := &appsv1.Deployment{
			Status: appsv1.DeploymentStatus{
				Replicas: 1,
			},
		}

		dogu := &core.Dogu{Name: "myDogu"}

		mResourceUpserter := newMockResourceUpserter(t)
		mResourceUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, assert.AnError)

		mDoguFetcher := newMockLocalDoguFetcher(t)
		mDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		mDoguInterface := newMockDoguInterface(t)
		mDoguInterface.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
			updatedStatus := f(dogu.Status)
			assert.Equal(t, "stopping", updatedStatus.Status)
			assert.False(t, updatedStatus.Stopped)

			return nil, nil
		})

		mDeploymentInterface := newMockDeploymentInterface(t)
		mDeploymentInterface.EXPECT().Get(testCtx, "myDogu", metav1.GetOptions{}).Return(deployment, nil)

		m := doguStartStopManager{
			resourceUpserter:    mResourceUpserter,
			doguFetcher:         mDoguFetcher,
			doguInterface:       mDoguInterface,
			deploymentInterface: mDeploymentInterface,
		}

		err := m.StartStopDogu(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to upsert deployment for starting/stopping dogu \"myDogu\":")
	})
}

func Test_doguNotYetStartedStoppedError(t *testing.T) {
	t.Run("should return correct error message for stopping", func(t *testing.T) {
		err := doguNotYetStartedStoppedError{doguName: "myDogu", stopped: true}

		assert.Equal(t, "the dogu \"myDogu\" has not yet been changed to its desired state: stopped", err.Error())
		assert.True(t, err.Requeue())
		assert.Equal(t, requeueWaitTimeout, err.GetRequeueTime())
	})

	t.Run("should return correct error message for starting", func(t *testing.T) {
		err := doguNotYetStartedStoppedError{doguName: "myDogu", stopped: false}

		assert.Equal(t, "the dogu \"myDogu\" has not yet been changed to its desired state: started", err.Error())
		assert.True(t, err.Requeue())
		assert.Equal(t, requeueWaitTimeout, err.GetRequeueTime())
	})

	t.Run("should return correct error message with nested error", func(t *testing.T) {
		err := doguNotYetStartedStoppedError{doguName: "myDogu", stopped: true, err: assert.AnError}

		assert.Equal(t, "error while starting/stopping dogu \"myDogu\": assert.AnError general error for testing", err.Error())
		assert.True(t, err.Requeue())
		assert.Equal(t, requeueWaitTimeout, err.GetRequeueTime())
	})
}
