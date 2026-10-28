package install

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3beta1ac "github.com/cloudogu/k8s-dogu-lib/v3/applyconfiguration/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeStatusStep reports every volume of a dogu, also of a stopped one, in status.volumeStatus.
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
	// add all claims of the currently deleted workloads, too
	volumes, err = vss.addUnmountedVolumes(ctx, doguResource.Namespace, rendered, volumes)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	previous := doguResource.Status.VolumeStatus
	statuses, err := updateVolumeStatus(ctx, vss.k8sClient, doguResource, rendered, volumes, nil)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}
	vss.reportResizedVolumes(ctx, doguResource, previous, statuses)

	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	return waitForResize(statuses)
}

// addUnmountedVolumes adds the live claims the chart renders but no workload mounts, e.g. while a resize recreates their StatefulSet.
func (vss *VolumeStatusStep) addUnmountedVolumes(ctx context.Context, namespace string, rendered []*unstructured.Unstructured, volumes []workloads.Volume) ([]workloads.Volume, error) {
	claims := &corev1.PersistentVolumeClaimList{}
	if err := vss.k8sClient.List(ctx, claims, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list PVCs in namespace %q: %w", namespace, err)
	}
	desiredSizes, err := pvc.DesiredSizes(rendered, namespace, claims.Items)
	if err != nil {
		return nil, fmt.Errorf("failed to read desired volume sizes: %w", err)
	}

	for _, claim := range claims.Items {
		_, isRendered := desiredSizes[client.ObjectKeyFromObject(&claim)]
		isMounted := slices.ContainsFunc(volumes, func(volume workloads.Volume) bool { return volume.PVC.Name == claim.Name })
		if isRendered && !isMounted {
			volumes = append(volumes, workloads.Volume{PVC: claim})
		}
	}

	return volumes, nil
}

func (vss *VolumeStatusStep) reportResizedVolumes(ctx context.Context, doguResource *v3beta1.Dogu, previous, statuses []v3beta1.VolumeStatus) {
	for _, status := range statuses {
		wasResizing := slices.ContainsFunc(previous, func(old v3beta1.VolumeStatus) bool {
			return old.ClaimName == status.ClaimName && old.State == v3beta1.VolumeStatusStateResizing
		})
		if wasResizing && status.State == v3beta1.VolumeStatusStateReady {
			logging.Info(ctx, "volume resized", "pvc", status.ClaimName, "size", status.Size.String())
			vss.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Volume %s resized to %s", status.ClaimName, status.Size.String()))
		}
	}
}

// updateVolumeStatus writes status.volumeStatus from the live volumes and reports failedClaims as ResizeFailed.
func updateVolumeStatus(ctx context.Context, k8sClient K8sClient, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, volumes []workloads.Volume, failedClaims []string) ([]v3beta1.VolumeStatus, error) {
	livePVCs := make([]corev1.PersistentVolumeClaim, 0, len(volumes))
	for _, volume := range volumes {
		livePVCs = append(livePVCs, volume.PVC)
	}
	desiredSizes, err := pvc.DesiredSizes(rendered, doguResource.Namespace, livePVCs)
	if err != nil {
		return nil, fmt.Errorf("failed to read desired volume sizes: %w", err)
	}

	statuses := make([]v3beta1.VolumeStatus, 0, len(volumes))
	for _, volume := range volumes {
		state := volumeState(&volume.PVC)
		if slices.Contains(failedClaims, volume.PVC.Name) {
			state = v3beta1.VolumeStatusStateResizeFailed
		}
		statuses = append(statuses, v3beta1.VolumeStatus{
			ClaimName:   volume.PVC.Name,
			Size:        volume.PVC.Status.Capacity.Storage().DeepCopy(),
			DesiredSize: desiredSizes[client.ObjectKeyFromObject(&volume.PVC)],
			State:       state,
			MountedBy:   volume.MountedBy,
		})
	}
	slices.SortFunc(statuses, func(a, b v3beta1.VolumeStatus) int {
		return strings.Compare(a.ClaimName, b.ClaimName)
	})

	if !equality.Semantic.DeepEqual(doguResource.Status.VolumeStatus, statuses) {
		doguResource.Status.VolumeStatus = statuses
		if err := k8sClient.Status().Update(ctx, doguResource); err != nil {
			return nil, fmt.Errorf("failed to update volume status: %w", err)
		}
	}

	return statuses, nil
}

const offlineExpansionHint = "if the storage only expands unmounted volumes, turn on offline volume expansion"

// waitForResize keeps the dogu in ResizingPVC until every rendered volume has reached its desired capacity.
func waitForResize(statuses []v3beta1.VolumeStatus) stepsv3.StepResult {
	var resizing, failed []string
	for _, status := range statuses {
		if status.DesiredSize.IsZero() {
			continue
		}
		switch status.State {
		case v3beta1.VolumeStatusStateResizing:
			resizing = append(resizing, status.ClaimName)
		case v3beta1.VolumeStatusStateResizeFailed:
			failed = append(failed, status.ClaimName)
		case v3beta1.VolumeStatusStateReady:
		}
	}

	if len(failed) > 0 {
		return stepsv3.RequeueWithError(fmt.Errorf("volumes failed to resize: %v; %s", failed, offlineExpansionHint), v3beta1.ReasonResizingPVC)
	}
	if len(resizing) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for volumes to reach their desired capacity: %v; %s", resizing, offlineExpansionHint))
	}

	return stepsv3.Continue()
}

func renderChart(ctx context.Context, chartService chartService, assembler valueAssembler, doguResource *v3beta1.Dogu) ([]*unstructured.Unstructured, error) {
	values, err := assembleValues(ctx, doguResource, chartService, assembler)
	if err != nil {
		return nil, err
	}

	rendered, err := chartService.Render(ctx, doguResource, values)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return rendered, nil
}

func volumeState(claim *corev1.PersistentVolumeClaim) v3beta1.VolumeStatusState {
	if pvc.HasTrueCondition(claim, corev1.PersistentVolumeClaimControllerResizeError, corev1.PersistentVolumeClaimNodeResizeError) {
		return v3beta1.VolumeStatusStateResizeFailed
	}

	if claim.Spec.Resources.Requests.Storage().Cmp(*claim.Status.Capacity.Storage()) > 0 ||
		pvc.HasTrueCondition(claim, corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimFileSystemResizePending) {
		return v3beta1.VolumeStatusStateResizing
	}

	return v3beta1.VolumeStatusStateReady
}
