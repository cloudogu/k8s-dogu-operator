package install

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VolumeStatusStep reports every volume of a dogu, also of a stopped one, in status.volumeStatus.
type VolumeStatusStep struct {
	chartService ChartService
	assembler    ValueAssembler
	discovery    VolumeDiscovery
	k8sClient    K8sClient
}

func NewVolumeStatusStep(chartService ChartService, assembler ValueAssembler, discovery VolumeDiscovery, k8sClient K8sClient) *VolumeStatusStep {
	return &VolumeStatusStep{
		chartService: chartService,
		assembler:    assembler,
		discovery:    discovery,
		k8sClient:    k8sClient,
	}
}

func (vss *VolumeStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	rendered, err := vss.renderChart(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	doguWorkloads, err := vss.discovery.Workloads(ctx, doguResource.Namespace, doguResource.Spec.Name)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to discover workloads: %w", err), v3beta1.ReasonInstalling)
	}

	volumes, err := vss.discovery.Volumes(ctx, doguResource.Namespace, doguWorkloads)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to discover volumes: %w", err), v3beta1.ReasonInstalling)
	}

	livePVCs := make([]corev1.PersistentVolumeClaim, 0, len(volumes))
	for _, volume := range volumes {
		livePVCs = append(livePVCs, volume.PVC)
	}
	desiredSizes, err := pvc.DesiredSizes(rendered, doguResource.Namespace, livePVCs)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to read desired volume sizes: %w", err), v3beta1.ReasonInstalling)
	}

	statuses := make([]v3beta1.VolumeStatus, 0, len(volumes))
	for _, volume := range volumes {
		statuses = append(statuses, v3beta1.VolumeStatus{
			ClaimName:   volume.PVC.Name,
			Size:        volume.PVC.Status.Capacity.Storage().DeepCopy(),
			DesiredSize: desiredSizes[client.ObjectKeyFromObject(&volume.PVC)],
			State:       volumeState(&volume.PVC),
			MountedBy:   volume.MountedBy,
		})
	}
	slices.SortFunc(statuses, func(a, b v3beta1.VolumeStatus) int {
		return strings.Compare(a.ClaimName, b.ClaimName)
	})

	if equality.Semantic.DeepEqual(doguResource.Status.VolumeStatus, statuses) {
		return stepsv3.Continue()
	}

	doguResource.Status.VolumeStatus = statuses
	if err := vss.k8sClient.Status().Update(ctx, doguResource); err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to update volume status: %w", err), v3beta1.ReasonInstalling)
	}

	return stepsv3.Continue()
}

func (vss *VolumeStatusStep) renderChart(ctx context.Context, doguResource *v3beta1.Dogu) ([]*unstructured.Unstructured, error) {
	metaValues, _, err := vss.chartService.DoguMetaValues(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu meta values: %w", err)
	}

	values, err := vss.assembler.Assemble(ctx, doguResource, metaValues)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble values: %w", err)
	}

	rendered, err := vss.chartService.Render(ctx, doguResource, values)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart: %w", err)
	}

	return rendered, nil
}

func volumeState(claim *corev1.PersistentVolumeClaim) v3beta1.VolumeStatusState {
	if hasTrueCondition(claim, corev1.PersistentVolumeClaimControllerResizeError, corev1.PersistentVolumeClaimNodeResizeError) {
		return v3beta1.VolumeStatusStateResizeFailed
	}

	if claim.Spec.Resources.Requests.Storage().Cmp(*claim.Status.Capacity.Storage()) > 0 ||
		hasTrueCondition(claim, corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimFileSystemResizePending) {
		return v3beta1.VolumeStatusStateResizing
	}

	return v3beta1.VolumeStatusStateReady
}

func hasTrueCondition(claim *corev1.PersistentVolumeClaim, conditionTypes ...corev1.PersistentVolumeClaimConditionType) bool {
	for _, condition := range claim.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && slices.Contains(conditionTypes, condition.Type) {
			return true
		}
	}

	return false
}
