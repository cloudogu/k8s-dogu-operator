package controllers

import (
	"context"
	"fmt"
	"time"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ChangeExportModeEventReason is the reason string for firing events for activating/deactivating the export mode.
	ChangeExportModeEventReason = "ChangeExportMode"
	// ErrorOnChangeExportModeEventReason is the error string for firing change export mode error events.
	ErrorOnChangeExportModeEventReason = "ErrChangeExportMode"
)

type exportModeNotYetChangedError struct {
	err                    error
	doguName               string
	desiredExportModeState bool
}

// Error built-in interface type is the conventional interface for
// representing an error condition, with the nil value representing no error.
func (e exportModeNotYetChangedError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("error while changing the export-mode of dogu %q: %v", e.doguName, e.err)
	}

	return fmt.Sprintf("the export-mode of dogu %q has not yet been changed to its desired state: %v", e.doguName, e.desiredExportModeState)
}

// Requeue returns true when the error should produce a requeue for the current dogu resource operation.
func (e exportModeNotYetChangedError) Requeue() bool {
	return true
}

// GetRequeueTime return the time to wait before the next reconciliation.
func (e exportModeNotYetChangedError) GetRequeueTime() time.Duration {
	return requeueWaitTimeout
}

type doguExportManager struct {
	doguClient       doguClient.DoguInterface
	podClient        podInterface
	resourceUpserter resource.ResourceUpserter
	doguFetcher      localDoguFetcher
	eventRecorder    record.EventRecorder
}

// NewDoguExportManager creates a new doguExportManager
func NewDoguExportManager(
	doguClient doguClient.DoguInterface,
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

func (dem *doguExportManager) shouldUpdateExportMode(ctx context.Context, doguResource *k8sv2.Dogu) (bool, error) {
	shouldExportModeBeActive := doguResource.Spec.ExportMode

	isExportModeActive, err := dem.isDeploymentInExportMode(ctx, doguResource.GetObjectKey())
	if err != nil {
		return true, fmt.Errorf("failed to check if deployment is in export-mode dogu %q: %w", doguResource.Name, err)
	}

	return shouldExportModeBeActive != isExportModeActive, nil
}

// UpdateExportMode activates/deactivates the export mode for the dogu
func (dem *doguExportManager) UpdateExportMode(ctx context.Context, doguResource *k8sv2.Dogu) error {
	logger := log.FromContext(ctx)

	shouldUpdate, err := dem.shouldUpdateExportMode(ctx, doguResource)
	if shouldUpdate || err != nil {
		if err != nil {
			logger.Error(err, "error while checking export-mode.")
		}

		if updateErr := dem.updateExportMode(ctx, doguResource); updateErr != nil {
			return updateErr
		}

		return exportModeNotYetChangedError{doguName: doguResource.Name, desiredExportModeState: doguResource.Spec.ExportMode, err: err}
	}

	logger.Info(fmt.Sprintf("The export-mode of dogu %q has changed to its desired state: %v", doguResource.Name, doguResource.Spec.ExportMode))
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
	podList, err := dem.podClient.List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", k8sv2.DoguLabelName, doguName.Name)})
	if err != nil {
		return false, fmt.Errorf("failed to get pods of deployment %q: %w", doguName, err)
	}

	exporterContainerName := resource.CreateExporterContainerName(doguName.Name)
	for _, pod := range podList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == exporterContainerName && containerStatus.Ready {
				return true, nil
			}
		}
	}

	return false, nil
}
