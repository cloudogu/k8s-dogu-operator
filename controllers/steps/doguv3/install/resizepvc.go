package install

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	flux "github.com/fluxcd/helm-controller/api/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizePVCStep grows the PVCs of a dogu to their rendered size and removes every workload and pod that mounts them,
// so that the resumed release recreates the workloads on the grown volumes.
type ResizePVCStep struct {
	chartService  ChartService
	assembler     ValueAssembler
	resizeChecker ResizeChecker
	discovery     VolumeDiscovery
	// apiReader reads uncached, so a lagging cache cannot hide a mounting pod or a resumed release.
	apiReader client.Reader
	k8sClient K8sClient
}

// NewResizePVCStep expects a discovery that reads through apiReader as well.
func NewResizePVCStep(chartService ChartService, assembler ValueAssembler, resizeChecker ResizeChecker, discovery VolumeDiscovery, apiReader client.Reader, k8sClient K8sClient) *ResizePVCStep {
	return &ResizePVCStep{
		chartService:  chartService,
		assembler:     assembler,
		resizeChecker: resizeChecker,
		discovery:     discovery,
		apiReader:     apiReader,
		k8sClient:     k8sClient,
	}
}

func (rps *ResizePVCStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	rendered, err := renderChart(ctx, rps.chartService, rps.assembler, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	checkResult, err := rps.resizeChecker.Check(ctx, doguResource.Namespace, rendered)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to check volumes: %w", err), v3beta1.ReasonResizingPVC)
	}

	affected, err := rps.affectedClaims(ctx, doguResource, checkResult)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(affected) == 0 {
		return stepsv3.Continue()
	}

	for _, request := range checkResult.ResizeRequests {
		if err := rps.patchStorageRequest(ctx, request); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
		}
	}

	deletedWorkloads, err := rps.deleteMountingWorkloads(ctx, doguResource, affected)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deletedWorkloads) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for workloads that mount resizing volumes to be deleted: %v", deletedWorkloads))
	}

	deletedPods, err := rps.deleteMountingPods(ctx, doguResource.Namespace, affected)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deletedPods) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for pods that mount resizing volumes to terminate: %v", deletedPods))
	}

	return stepsv3.Continue()
}

// affectedClaims returns the claims to grow and, while the release is still suspended, those whose growth is in progress.
// Once the release is resumed, the workloads mounting an in-progress claim are already the recreated ones. We need to
// resume in the inProgress-State, because many storage-provider only resize mounted PVCs.
func (rps *ResizePVCStep) affectedClaims(ctx context.Context, doguResource *v3beta1.Dogu, checkResult pvc.CheckResult) ([]string, error) {
	var claims []string
	for _, request := range checkResult.ResizeRequests {
		claims = append(claims, request.PVC.Name)
	}
	if len(checkResult.InProgress) == 0 {
		return claims, nil
	}

	release := &flux.HelmRelease{}
	err := rps.apiReader.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		return claims, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get HelmRelease %q: %w", doguResource.Spec.Name, err)
	}
	if release.Spec.Suspend {
		for _, key := range checkResult.InProgress {
			claims = append(claims, key.Name)
		}
	}

	return claims, nil
}

func (rps *ResizePVCStep) patchStorageRequest(ctx context.Context, request pvc.ResizeRequest) error {
	claim := &corev1.PersistentVolumeClaim{}
	if err := rps.k8sClient.Get(ctx, request.PVC, claim); err != nil {
		return fmt.Errorf("failed to get PVC %q: %w", request.PVC, err)
	}

	patch := client.MergeFrom(claim.DeepCopy())
	claim.Spec.Resources.Requests[corev1.ResourceStorage] = request.Desired.DeepCopy()
	if err := rps.k8sClient.Patch(ctx, claim, patch); err != nil {
		return fmt.Errorf("failed to grow PVC %q to %s: %w", request.PVC, request.Desired.String(), err)
	}

	return nil
}

func (rps *ResizePVCStep) deleteMountingWorkloads(ctx context.Context, doguResource *v3beta1.Dogu, claims []string) ([]string, error) {
	doguWorkloads, err := rps.discovery.Workloads(ctx, doguResource.Namespace, doguResource.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workloads: %w", err)
	}
	volumes, err := rps.discovery.Volumes(ctx, doguResource.Namespace, doguWorkloads)
	if err != nil {
		return nil, fmt.Errorf("failed to discover volumes: %w", err)
	}

	var deleted []string
	for _, volume := range volumes {
		if !slices.Contains(claims, volume.PVC.Name) {
			continue
		}
		for _, ref := range volume.MountedBy {
			name := ref.Kind + "/" + ref.Name
			if slices.Contains(deleted, name) {
				continue
			}
			workload, propagation := findWorkload(doguWorkloads, ref)
			if workload == nil {
				continue
			}
			uid := workload.GetUID()
			err := rps.k8sClient.Delete(ctx, workload, client.Preconditions{UID: &uid}, client.PropagationPolicy(propagation))
			if err != nil && !apierrors.IsNotFound(err) {
				return nil, fmt.Errorf("failed to delete %s: %w", name, err)
			}
			deleted = append(deleted, name)
		}
	}

	return deleted, nil
}

// findWorkload returns the workload a volume reference points to; StatefulSets are orphaned so their PVCs survive.
func findWorkload(doguWorkloads workloads.Workloads, ref corev1.TypedLocalObjectReference) (client.Object, metav1.DeletionPropagation) {
	switch ref.Kind {
	case "Deployment":
		for i := range doguWorkloads.Deployments {
			if doguWorkloads.Deployments[i].Name == ref.Name {
				return &doguWorkloads.Deployments[i], metav1.DeletePropagationBackground
			}
		}
	case "DaemonSet":
		for i := range doguWorkloads.DaemonSets {
			if doguWorkloads.DaemonSets[i].Name == ref.Name {
				return &doguWorkloads.DaemonSets[i], metav1.DeletePropagationBackground
			}
		}
	case "StatefulSet":
		for i := range doguWorkloads.StatefulSets {
			if doguWorkloads.StatefulSets[i].Name == ref.Name {
				return &doguWorkloads.StatefulSets[i], metav1.DeletePropagationOrphan
			}
		}
	}

	return nil, ""
}

// deleteMountingPods finds running pods by claim name; they carry no dogu label and may have outlived their workload.
func (rps *ResizePVCStep) deleteMountingPods(ctx context.Context, namespace string, claims []string) ([]string, error) {
	pods := &corev1.PodList{}
	if err := rps.apiReader.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %q: %w", namespace, err)
	}

	var deleted []string
	for i := range pods.Items {
		pod := &pods.Items[i]
		// skip finished pods to keep the logs
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed || !podMountsAny(pod, claims) {
			continue
		}
		if err := rps.k8sClient.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete pod %q: %w", pod.Name, err)
		}
		deleted = append(deleted, pod.Name)
	}

	return deleted, nil
}

func podMountsAny(pod *corev1.Pod, claims []string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && slices.Contains(claims, volume.PersistentVolumeClaim.ClaimName) {
			return true
		}
	}

	return false
}
