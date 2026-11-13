package install

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3beta1ac "github.com/cloudogu/k8s-dogu-lib/v3/applyconfiguration/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeStatusStep reports every volume of a dogu, also of a stopped one, in status.volumeStatus and waits until every volume has its desired capacity.
type VolumeStatusStep struct {
	chartService  chartService
	assembler     valueAssembler
	discovery     VolumeDiscovery
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewVolumeStatusStep(chartService chartService, assembler valueAssembler, discovery VolumeDiscovery, k8sClient K8sClient, recorder EventRecorder) *VolumeStatusStep {
	return &VolumeStatusStep{
		chartService:  chartService,
		assembler:     assembler,
		discovery:     discovery,
		k8sClient:     k8sClient,
		eventRecorder: recorder,
	}
}

func (vss *VolumeStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	rendered, err := renderChart(ctx, vss.chartService, vss.assembler, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	doguWorkloads, err := vss.discovery.Workloads(ctx, doguResource.Namespace, doguResource.Spec.Name)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to discover workloads: %w", err), v3beta1.ReasonInstalling)
	}

	// find all mounted claims, even existing ones that the chart does not include
	volumes, err := vss.discovery.Volumes(ctx, doguResource.Namespace, doguWorkloads)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to discover volumes: %w", err), v3beta1.ReasonInstalling)
	}

	previous := doguResource.Status.VolumeStatus
	statuses, err := updateVolumeStatus(ctx, vss.k8sClient, doguResource, rendered, volumes)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}
	vss.reportResizedVolumes(ctx, doguResource, previous, statuses)

	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	return waitForResizeCompletion(statuses)
}

func (vss *VolumeStatusStep) reportResizedVolumes(ctx context.Context, doguResource *v3beta1.Dogu, previous, statuses []v3beta1.VolumeStatus) {
	for _, status := range statuses {
		// a resize also counts when capacity grew between observations without a Resizing status
		resized := slices.ContainsFunc(previous, func(old v3beta1.VolumeStatus) bool {
			return old.ClaimName == status.ClaimName &&
				(old.State == v3beta1.VolumeStatusStateResizing || old.Size.Cmp(status.Size) < 0)
		})
		if resized && status.State == v3beta1.VolumeStatusStateReady {
			logging.Info(ctx, "volume resized", "pvc", status.ClaimName, "size", status.Size.String())
			vss.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Volume %s resized to %s", status.ClaimName, status.Size.String()))
		}
	}
}

// updateVolumeStatus writes status.volumeStatus from the live volumes.
func updateVolumeStatus(ctx context.Context, k8sClient K8sClient, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, volumes []pvc.Volume) ([]v3beta1.VolumeStatus, error) {
	statuses, err := pvc.VolumeStatuses(rendered, doguResource.Namespace, volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume statuses: %w", err)
	}

	if !equality.Semantic.DeepEqual(doguResource.Status.VolumeStatus, statuses) {
		if err := applyVolumeStatus(ctx, k8sClient, doguResource, statuses); err != nil {
			return nil, err
		}
	}

	return statuses, nil
}

// applyVolumeStatus owns status.volumeStatus via server-side apply, so it does not conflict with other status writers.
func applyVolumeStatus(ctx context.Context, k8sClient K8sClient, doguResource *v3beta1.Dogu, statuses []v3beta1.VolumeStatus) error {
	configurations := make([]*v3beta1ac.VolumeStatusApplyConfiguration, 0, len(statuses))
	for _, status := range statuses {
		configuration := v3beta1ac.VolumeStatus().
			WithClaimName(status.ClaimName).
			WithState(status.State).
			WithMountedBy(status.MountedBy...)
		// omit zero default-values
		if !status.Size.IsZero() {
			configuration.WithSize(status.Size)
		}
		if !status.DesiredSize.IsZero() {
			configuration.WithDesiredSize(status.DesiredSize)
		}
		configurations = append(configurations, configuration)
	}
	doguConfiguration := v3beta1ac.Dogu(doguResource.Name, doguResource.Namespace).
		WithStatus(v3beta1ac.DoguStatus().WithVolumeStatus(configurations...))

	if err := k8sClient.Status().Apply(ctx, doguConfiguration, client.FieldOwner(volumeStatusFieldOwner), client.ForceOwnership); err != nil {
		return fmt.Errorf("failed to update volume status: %w", err)
	}

	// SSA does not update the resource -> we have to do it so that later update calls do not get conflicts
	doguResource.Status.VolumeStatus = statuses
	if doguConfiguration.ResourceVersion != nil {
		doguResource.ResourceVersion = *doguConfiguration.ResourceVersion
	}
	return nil
}

const volumeStatusFieldOwner = "k8s-dogu-operator-volume-status-step"

const offlineExpansionHint = "if the storage only expands unmounted volumes, turn on offline volume expansion"

// waitForResizeCompletion requeues until no volume is below its desired capacity.
func waitForResizeCompletion(statuses []v3beta1.VolumeStatus) stepsv3.StepResult {
	var failed, resizing []string
	for _, status := range statuses {
		switch {
		case status.State == v3beta1.VolumeStatusStateResizeFailed:
			failed = append(failed, status.ClaimName)
		case status.State == v3beta1.VolumeStatusStateResizing || status.Size.Cmp(status.DesiredSize) < 0:
			resizing = append(resizing, status.ClaimName)
		}
	}

	if len(failed) > 0 {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to resize volumes: %v", failed), v3beta1.ReasonResizingPVC)
	}
	if len(resizing) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for volumes to reach their desired capacity: %v; %s", resizing, offlineExpansionHint))
	}

	return stepsv3.Continue()
}
