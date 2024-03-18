package controllers

import (
	"context"
	"errors"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	mocks2 "github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	scalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_doguStartStopManager_CheckStarted(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		rolledOutDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cas",
				Namespace: "test",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				ReadyReplicas:     1,
				UpdatedReplicas:   1,
				AvailableReplicas: 1,
			},
		}
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(rolledOutDeployment, nil)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv1.Dogu, modifyStatusFn func(k8sv1.DoguStatus) k8sv1.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, k8sv1.DoguStatusInstalled, status.Status)
				assert.Equal(t, false, status.Stopped)
			})

		podInterfaceMock := mocks.NewPodInterface(t)
		podList := &v1.PodList{Items: []v1.Pod{{Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "cas"}}}}}}
		podInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=cas"}).Return(podList, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStarted(testCtx, dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on deployment get error", func(t *testing.T) {
		// given
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(nil, assert.AnError)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)
		podInterfaceMock := mocks.NewPodInterface(t)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStarted(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to start dogu \"test/cas\": failed to get deployment \"test/cas\"")
	})

	t.Run("should return deployment not yet scaled error if deployment is not rolled out", func(t *testing.T) {
		// given
		rolledOutDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cas",
				Namespace: "test",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				ReadyReplicas:     0,
				UpdatedReplicas:   0,
				AvailableReplicas: 0,
			},
		}
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(rolledOutDeployment, nil)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)

		podInterfaceMock := mocks.NewPodInterface(t)
		podList := &v1.PodList{Items: []v1.Pod{{Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "cas"}}}}}}
		podInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=cas"}).Return(podList, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStarted(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "the deployment of dogu \"test/cas\" has not yet been scaled to its desired number of replicas")
		var requeueError deploymentNotYetScaledError
		errors.As(err, &requeueError)
	})
}

func Test_doguStartStopManager_CheckStopped(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		scaledDownDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cas",
				Namespace: "test",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          0,
				ReadyReplicas:     0,
				UpdatedReplicas:   0,
				AvailableReplicas: 0,
			},
		}
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(scaledDownDeployment, nil)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv1.Dogu, modifyStatusFn func(k8sv1.DoguStatus) k8sv1.DoguStatus, opts metav1.UpdateOptions) {
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, k8sv1.DoguStatusInstalled, status.Status)
				assert.Equal(t, true, status.Stopped)
			})

		podInterfaceMock := mocks.NewPodInterface(t)
		podList := &v1.PodList{Items: []v1.Pod{{Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "cas"}}}}}}
		podInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=cas"}).Return(podList, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStopped(testCtx, dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("should return error on deployment get error", func(t *testing.T) {
		// given
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(nil, assert.AnError)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)
		podInterfaceMock := mocks.NewPodInterface(t)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStopped(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to stop dogu \"test/cas\": failed to get deployment \"test/cas\"")
	})

	t.Run("should return deployment not yet scaled error if deployment is not rolled out", func(t *testing.T) {
		// given
		rolledOutDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cas",
				Namespace: "test",
			},
			Status: appsv1.DeploymentStatus{
				Replicas:          1,
				ReadyReplicas:     0,
				UpdatedReplicas:   0,
				AvailableReplicas: 0,
			},
		}
		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		deploymentInterfaceMock.EXPECT().Get(testCtx, "cas", metav1.GetOptions{}).Return(rolledOutDeployment, nil)

		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}
		doguInterfaceMock := mocks2.NewDoguInterface(t)

		podInterfaceMock := mocks.NewPodInterface(t)
		podList := &v1.PodList{Items: []v1.Pod{{Status: v1.PodStatus{ContainerStatuses: []v1.ContainerStatus{{Name: "cas"}}}}}}
		podInterfaceMock.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=cas"}).Return(podList, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock, podInterface: podInterfaceMock}

		// when
		err := sut.CheckStopped(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "the deployment of dogu \"test/cas\" has not yet been scaled to its desired number of replicas")
		var requeueError deploymentNotYetScaledError
		errors.As(err, &requeueError)
	})
}

func Test_deploymentNotYetScaledError(t *testing.T) {
	t.Run("deployment not yet scaled error should requeue", func(t *testing.T) {
		assert.True(t, deploymentNotYetScaledError{}.Requeue())
	})

	t.Run("deployment not yet scaled error should have requeue time", func(t *testing.T) {
		assert.Equal(t, 5*time.Second, deploymentNotYetScaledError{}.GetRequeueTime())
	})
}

func Test_doguStartStopManager_StartDogu(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}

		doguInterfaceMock := mocks2.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv1.Dogu, modifyStatusFn func(k8sv1.DoguStatus) k8sv1.DoguStatus, opts metav1.UpdateOptions) {
				oldStopped := dogu.Status.Stopped
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, k8sv1.DoguStatusStarting, status.Status)
				assert.Equal(t, oldStopped, status.Stopped)
			})

		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		scale := &scalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}, Spec: scalingv1.ScaleSpec{Replicas: 1}}
		deploymentInterfaceMock.EXPECT().UpdateScale(testCtx, "cas", scale, metav1.UpdateOptions{}).Return(nil, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock}

		// when
		err := sut.StartDogu(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "the deployment of dogu \"test/cas\" has not yet been scaled to its desired number of replicas")
		var requeueError deploymentNotYetScaledError
		errors.As(err, &requeueError)
	})
}

func Test_doguStartStopManager_StopDogu(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := &k8sv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}}

		doguInterfaceMock := mocks2.NewDoguInterface(t)
		doguInterfaceMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).
			Run(func(ctx context.Context, dogu *k8sv1.Dogu, modifyStatusFn func(k8sv1.DoguStatus) k8sv1.DoguStatus, opts metav1.UpdateOptions) {
				oldStopped := dogu.Status.Stopped
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, k8sv1.DoguStatusStopping, status.Status)
				assert.Equal(t, oldStopped, status.Stopped)
			})

		deploymentInterfaceMock := mocks.NewDeploymentInterface(t)
		scale := &scalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: "cas", Namespace: "test"}, Spec: scalingv1.ScaleSpec{Replicas: 0}}
		deploymentInterfaceMock.EXPECT().UpdateScale(testCtx, "cas", scale, metav1.UpdateOptions{}).Return(nil, nil)

		sut := doguStartStopManager{doguInterface: doguInterfaceMock, deploymentInterface: deploymentInterfaceMock}

		// when
		err := sut.StopDogu(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "the deployment of dogu \"test/cas\" has not yet been scaled to its desired number of replicas")
		var requeueError deploymentNotYetScaledError
		errors.As(err, &requeueError)
	})
}
