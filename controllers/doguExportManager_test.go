package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_doguExportManager_UpdateExportMode(t *testing.T) {
	t.Run("should update deployment when export-mode changes", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		dogu := &core.Dogu{}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: false}}}},
		}}

		deploymentList := &appsv1.DeploymentList{Items: []appsv1.Deployment{
			{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: nil}}}},
		}}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil)

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		mockDeploymentClient := newMockDeploymentInterface(t)
		mockDeploymentClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(deploymentList, nil)

		mockDoguFetcher := newMockLocalDoguFetcher(t)
		mockDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		mockUpserter := newMockResourceUpserter(t)
		mockUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, nil)

		dem := &doguExportManager{
			doguClient:       mockDoguClient,
			podClient:        mockPodClient,
			deploymentClient: mockDeploymentClient,
			doguFetcher:      mockDoguFetcher,
			resourceUpserter: mockUpserter,
		}

		err := dem.UpdateExportMode(testCtx, doguResource)

		require.Error(t, err)
		require.ErrorIs(t, err, exportModeNotYetChangedError{doguName: "myDogu", desiredExportModeState: true})
		assert.Equal(t, "the export-mode of dogu \"myDogu\" has not yet been changed to its desired state: true", err.Error())
		assert.Equal(t, requeueWaitTimeout, err.(exportModeNotYetChangedError).GetRequeueTime())
		assert.True(t, err.(exportModeNotYetChangedError).Requeue())
	})

	t.Run("reconcile when could not get current state of export-mode", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		mockDoguClient := newMockDoguInterface(t)

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(nil, assert.AnError)

		mockDoguFetcher := newMockLocalDoguFetcher(t)
		mockUpserter := newMockResourceUpserter(t)

		dem := &doguExportManager{
			doguClient:       mockDoguClient,
			podClient:        mockPodClient,
			doguFetcher:      mockDoguFetcher,
			resourceUpserter: mockUpserter,
		}

		err := dem.UpdateExportMode(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error while changing the export-mode of dogu \"myDogu\": failed to check if deployment is in export-mode dogu \"myDogu\": failed to get pods of deployment \"/myDogu\":")
		assert.Equal(t, requeueWaitTimeout, err.(exportModeNotYetChangedError).GetRequeueTime())
		assert.True(t, err.(exportModeNotYetChangedError).Requeue())

		mockDoguFetcher.AssertNotCalled(t, "FetchInstalled")
		mockUpserter.AssertNotCalled(t, "UpsertDoguDeployment")
	})

	t.Run("should fail to update deployment when export-mode changes on error updating status", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: false}}}},
		}}

		deploymentList := &appsv1.DeploymentList{Items: []appsv1.Deployment{
			{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: nil}}}},
		}}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		mockDeploymentClient := newMockDeploymentInterface(t)
		mockDeploymentClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(deploymentList, nil)

		dem := &doguExportManager{
			doguClient:       mockDoguClient,
			podClient:        mockPodClient,
			deploymentClient: mockDeploymentClient,
		}

		err := dem.UpdateExportMode(testCtx, doguResource)
		require.Error(t, err)

		var oErr exportModeNotYetChangedError
		assert.True(t, errors.As(err, &oErr))
		assert.True(t, oErr.Requeue())
		assert.ErrorIs(t, oErr.err, assert.AnError)
	})

	t.Run("should not update deployment when export-mode already in desired state", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		//dogu := &core.Dogu{}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: true}}}},
		}}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil)

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		dem := &doguExportManager{
			doguClient: mockDoguClient,
			podClient:  mockPodClient,
		}

		err := dem.UpdateExportMode(testCtx, doguResource)
		require.NoError(t, err)
	})

	t.Run("should not update deployment when export-mode already set in deployment spec", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: false}}}},
		}}

		deploymentList := &appsv1.DeploymentList{Items: []appsv1.Deployment{
			{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "myDogu-exporter"}}}}}},
		}}

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		mockDeploymentClient := newMockDeploymentInterface(t)
		mockDeploymentClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(deploymentList, nil)

		dem := &doguExportManager{
			podClient:        mockPodClient,
			deploymentClient: mockDeploymentClient,
		}

		err := dem.UpdateExportMode(testCtx, doguResource)

		require.Error(t, err)
		require.ErrorIs(t, err, exportModeNotYetChangedError{doguName: "myDogu", desiredExportModeState: true})
		assert.Equal(t, "the export-mode of dogu \"myDogu\" has not yet been changed to its desired state: true", err.Error())
		assert.Equal(t, requeueWaitTimeout, err.(exportModeNotYetChangedError).GetRequeueTime())
		assert.True(t, err.(exportModeNotYetChangedError).Requeue())
		assert.Nil(t, err.(exportModeNotYetChangedError).err)
	})
}

func Test_doguExportManager_shouldUpdateExportMode(t *testing.T) {
	t.Run("should update export-mode changes", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: false}}}},
		}}

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		dem := &doguExportManager{
			podClient: mockPodClient,
		}

		result, err := dem.shouldUpdateExportMode(testCtx, doguResource)
		require.NoError(t, err)
		require.True(t, result)
	})

	t.Run("should not update export-mode", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		podList := &corev1.PodList{Items: []corev1.Pod{
			{Status: corev1.PodStatus{
				Phase:             corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{Name: "myDogu-exporter", Ready: true}}}},
		}}

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(podList, nil)

		dem := &doguExportManager{
			podClient: mockPodClient,
		}

		result, err := dem.shouldUpdateExportMode(testCtx, doguResource)
		require.NoError(t, err)
		require.False(t, result)
	})

	t.Run("should update export-mode for error getting pods", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		mockPodClient := newMockPodInterface(t)
		mockPodClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(nil, assert.AnError)

		dem := &doguExportManager{
			podClient: mockPodClient,
		}

		result, err := dem.shouldUpdateExportMode(testCtx, doguResource)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if deployment is in export-mode dogu \"myDogu\": failed to get pods of deployment \"/myDogu\":")
		assert.True(t, result)
	})
}

func Test_doguExportManager_updateExportMode(t *testing.T) {
	t.Run("should fail to update deployment on error getting dogu", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		dogu := &core.Dogu{}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil)

		mockDoguFetcher := newMockLocalDoguFetcher(t)
		mockDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, assert.AnError)

		deploymentList := &appsv1.DeploymentList{Items: []appsv1.Deployment{
			//{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "myDogu-exporter"}}}}}},
			{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: nil}}}},
		}}

		mockDeploymentClient := newMockDeploymentInterface(t)
		mockDeploymentClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(deploymentList, nil)

		dem := &doguExportManager{
			doguClient:       mockDoguClient,
			doguFetcher:      mockDoguFetcher,
			deploymentClient: mockDeploymentClient,
		}

		err := dem.updateExportMode(testCtx, doguResource)
		require.Error(t, err)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get local descriptor for dogu \"myDogu\":")
	})

	t.Run("should fail to update deployment on error upserting deployment", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		dogu := &core.Dogu{}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil)

		mockDoguFetcher := newMockLocalDoguFetcher(t)
		mockDoguFetcher.EXPECT().FetchInstalled(testCtx, doguResource.GetSimpleDoguName()).Return(dogu, nil)

		deploymentList := &appsv1.DeploymentList{Items: []appsv1.Deployment{
			//{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "myDogu-exporter"}}}}}},
			{Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: nil}}}},
		}}

		mockDeploymentClient := newMockDeploymentInterface(t)
		mockDeploymentClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: "dogu.name=myDogu"}).Return(deploymentList, nil)

		mockUpserter := newMockResourceUpserter(t)
		mockUpserter.EXPECT().UpsertDoguDeployment(testCtx, doguResource, dogu, mock.Anything).Return(nil, assert.AnError)

		dem := &doguExportManager{
			doguClient:       mockDoguClient,
			doguFetcher:      mockDoguFetcher,
			deploymentClient: mockDeploymentClient,
			resourceUpserter: mockUpserter,
		}

		err := dem.updateExportMode(testCtx, doguResource)
		require.Error(t, err)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to upsert deployment for export-mode change for dogu \"myDogu\":")
	})
}

func Test_doguExportManager_updateStatusWithRetry(t *testing.T) {
	t.Run("should update status", func(t *testing.T) {
		doguResource := &doguv2.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "myDogu"},
			Spec:       doguv2.DoguSpec{ExportMode: true},
		}

		mockDoguClient := newMockDoguInterface(t)
		mockDoguClient.EXPECT().UpdateStatusWithRetry(testCtx, doguResource, mock.Anything, metav1.UpdateOptions{}).Run(func(ctx context.Context, dogu *doguv2.Dogu, modifyStatusFn func(doguv2.DoguStatus) doguv2.DoguStatus, opts metav1.UpdateOptions) {
			status := doguv2.DoguStatus{Status: "foo", ExportMode: false}
			modifiedStatus := modifyStatusFn(status)

			assert.Equal(t, "testPhase", modifiedStatus.Status)
			assert.True(t, modifiedStatus.ExportMode)
		}).Return(nil, nil)

		dem := &doguExportManager{
			doguClient: mockDoguClient,
		}

		err := dem.updateStatusWithRetry(testCtx, doguResource, "testPhase", true)
		require.NoError(t, err)
	})
}
