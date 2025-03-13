package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
)

const (
	ChangeExportModeEventReason        = "ChangeExportMode"
	ErrorOnChangeExportModeEventReason = "ErrChangeExportMode"
)

type exportModeNotYetChangedError struct {
	doguName               string
	desiredExportModeState bool
}

func (e exportModeNotYetChangedError) Error() string {
	return fmt.Sprintf("the export-mode of dogu %q has not yet been changed to its desired state: %v", e.doguName, e.desiredExportModeState)
}

func (e exportModeNotYetChangedError) Requeue() bool {
	return true
}

func (e exportModeNotYetChangedError) GetRequeueTime() time.Duration {
	return requeueWaitTimeout
}

type doguExportManager struct {
	doguClient       ecoSystem.DoguInterface
	podClient        podInterface
	resourceUpserter resource.ResourceUpserter
	doguFetcher      localDoguFetcher
	eventRecorder    record.EventRecorder
}

func NewDoguExportManager(
	doguClient ecoSystem.DoguInterface,
	podClient podInterface,
	resourceUpserter resource.ResourceUpserter,
	doguFetcher localDoguFetcher,
	eventRecorder record.EventRecorder,
) *doguExportManager {
	return &doguExportManager{
		doguClient:       doguClient,
		podClient:        podClient,
		resourceUpserter: resourceUpserter,
		doguFetcher:      doguFetcher,
		eventRecorder:    eventRecorder,
	}
}

func (dem *doguExportManager) shouldUpdateExportMode(ctx context.Context, doguResource *k8sv2.Dogu) bool {
	logger := log.FromContext(ctx)

	shouldExportModeBeActive := doguResource.Spec.ExportMode

	isExportModeActive, err := dem.isDeploymentInExportMode(ctx, doguResource.GetObjectKey())
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to check if deployment is in export-mode dogu %q", doguResource.Name))
		return true
	}

	return shouldExportModeBeActive != isExportModeActive
}

func (dem *doguExportManager) UpdateExportMode(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)

	if dem.shouldUpdateExportMode(ctx, doguResource) {
		if err := dem.updateExportMode(ctx, doguResource); err != nil {
			return err
		}

		return exportModeNotYetChangedError{doguName: doguResource.Name, desiredExportModeState: doguResource.Spec.ExportMode}
	}

	logger.Info(fmt.Sprintf("the export-mode of dogu %q has changed to its desired state: %v", doguResource.Name, doguResource.Spec.ExportMode))
	return dem.updateStatusWithRetry(ctx, doguResource, k8sv2.DoguStatusInstalled, doguResource.Spec.ExportMode)
}

func (dem *doguExportManager) updateExportMode(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)

	if err := dem.updateStatusWithRetry(ctx, doguResource, k8sv2.DoguStatusChangingExportMode, doguResource.Status.ExportMode); err != nil {
		return err
	}

	logger.Info("Getting local dogu descriptor...")
	dogu, err := dem.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err)
	}

	logger.Info("Upserting deployment...")
	_, err = dem.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment for export-mode change for dogu %q: %w", doguResource.Name, err)
	}

	return nil
}

func (dem *doguExportManager) updateStatusWithRetry(ctx context.Context, doguResource *k8sv2.Dogu, phase string, activated bool) error {
	_, err := dem.doguClient.UpdateStatusWithRetry(ctx, doguResource, func(status k8sv2.DoguStatus) k8sv2.DoguStatus {
		status.Status = phase
		status.ExportMode = activated
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %q to %q: %w", doguResource.Name, phase, err)
	}

	return nil
}

func (dem *doguExportManager) isDeploymentInExportMode(ctx context.Context, doguName types.NamespacedName) (bool, error) {
	logrus.Info(fmt.Sprintf("check export-mode status for deployment %s", doguName))

	podList, getErr := dem.podClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("dogu.name=%s", doguName.Name)})
	if getErr != nil {
		return false, fmt.Errorf("failed to get pods of deployment %q: %w", doguName, getErr)
	}

	exporterContainerName := fmt.Sprintf("%s-exporter", doguName.Name)
	for _, pod := range podList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == exporterContainerName && containerStatus.Ready {
				return true, nil
			}
		}
	}

	return false, nil
}
