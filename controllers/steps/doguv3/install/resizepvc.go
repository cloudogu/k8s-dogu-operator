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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizePVCStep grows the PVCs of a dogu to their rendered size and replaces StatefulSets with outdated claim templates.
// With offline expansion it also removes every workload and pod that mounts a growing PVC, so that the resumed release
// recreates the workloads on the grown volumes.
type ResizePVCStep struct {
	chartService  ChartService
	assembler     ValueAssembler
	resizeChecker ResizeChecker
	discovery     VolumeDiscovery
	// apiReader reads uncached, so a lagging cache cannot hide a mounting pod or a resumed release.
	apiReader        client.Reader
	k8sClient        K8sClient
	offlineExpansion bool
}

// NewResizePVCStep expects a discovery that reads through apiReader as well.
func NewResizePVCStep(chartService ChartService, assembler ValueAssembler, resizeChecker ResizeChecker, discovery VolumeDiscovery, apiReader client.Reader, k8sClient K8sClient, offlineExpansion bool) *ResizePVCStep {
	return &ResizePVCStep{
		chartService:     chartService,
		assembler:        assembler,
		resizeChecker:    resizeChecker,
		discovery:        discovery,
		apiReader:        apiReader,
		k8sClient:        k8sClient,
		offlineExpansion: offlineExpansion,
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

	if rps.offlineExpansion {
		return rps.resizeOffline(ctx, doguResource, rendered, checkResult)
	}
	return rps.resizeOnline(ctx, doguResource, rendered, checkResult)
}

// resizeOnline grows the claims while their workloads and pods keep running.
func (rps *ResizePVCStep) resizeOnline(ctx context.Context, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, checkResult pvc.CheckResult) stepsv3.StepResult {
	if len(checkResult.ResizeRequests) == 0 && len(checkResult.OutdatedStatefulSets) == 0 {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, result := rps.growClaims(ctx, doguResource, rendered, checkResult.ResizeRequests)
	if !result.Continue {
		return result
	}

	return rps.replaceWorkloads(ctx, doguWorkloads, volumes, nil, checkResult.OutdatedStatefulSets)
}

// resizeOffline detaches the claims from all workloads and pods, for storage that only expands unmounted volumes.
func (rps *ResizePVCStep) resizeOffline(ctx context.Context, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, checkResult pvc.CheckResult) stepsv3.StepResult {
	affected, err := rps.affectedClaims(ctx, doguResource, checkResult)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(affected) == 0 && len(checkResult.OutdatedStatefulSets) == 0 {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, err := rps.discover(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	failedClaims, foreignOwners := foreignOwnedClaims(doguWorkloads, volumes, affected)
	if len(failedClaims) > 0 {
		if _, err := updateVolumeStatus(ctx, rps.k8sClient, doguResource, rendered, volumes, failedClaims); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
		}
		return stepsv3.Abort(v3beta1.ReasonResizingPVC, fmt.Sprintf("cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: %v", foreignOwners))
	}

	var result stepsv3.StepResult
	doguWorkloads, volumes, result = rps.growClaims(ctx, doguResource, rendered, checkResult.ResizeRequests)
	if !result.Continue {
		return result
	}

	if result = rps.replaceWorkloads(ctx, doguWorkloads, volumes, affected, checkResult.OutdatedStatefulSets); !result.Continue {
		return result
	}

	return rps.deleteMountingPods(ctx, doguResource.Namespace, affected)
}

// replaceWorkloads deletes the workloads mounting claims plus the outdated StatefulSets and waits until they are gone.
func (rps *ResizePVCStep) replaceWorkloads(ctx context.Context, doguWorkloads workloads.Workloads, volumes []workloads.Volume, claims []string, outdatedStatefulSets []types.UID) stepsv3.StepResult {
	deletedWorkloads, err := rps.deleteWorkloadsToRecreate(ctx, doguWorkloads, volumes, claims, outdatedStatefulSets)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deletedWorkloads) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for workloads to be deleted and recreated by the release: %v", deletedWorkloads))
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

// growClaims grows and reports the volume claims. It aborts if the cluster rejects a claim, before any workload is deleted.
func (rps *ResizePVCStep) growClaims(ctx context.Context, doguResource *v3beta1.Dogu, rendered []*unstructured.Unstructured, requests []pvc.ResizeRequest) (workloads.Workloads, []workloads.Volume, stepsv3.StepResult) {
	var rejectedClaims []string
	var rejection error
	for _, request := range requests {
		err := rps.patchStorageRequest(ctx, request)
		// Stop and report if resizing is not allowed
		if apierrors.IsForbidden(err) || apierrors.IsInvalid(err) {
			rejectedClaims, rejection = []string{request.PVC.Name}, err
			break
		}
		if err != nil {
			return workloads.Workloads{}, nil, stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
		}
	}

	doguWorkloads, volumes, err := rps.discover(ctx, doguResource)
	if err != nil {
		return workloads.Workloads{}, nil, stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if _, err := updateVolumeStatus(ctx, rps.k8sClient, doguResource, rendered, volumes, rejectedClaims); err != nil {
		return workloads.Workloads{}, nil, stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if rejection != nil {
		return workloads.Workloads{}, nil, stepsv3.Abort(v3beta1.ReasonResizingPVC, rejection.Error())
	}

	return doguWorkloads, volumes, stepsv3.Continue()
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

func (rps *ResizePVCStep) discover(ctx context.Context, doguResource *v3beta1.Dogu) (workloads.Workloads, []workloads.Volume, error) {
	doguWorkloads, err := rps.discovery.Workloads(ctx, doguResource.Namespace, doguResource.Spec.Name)
	if err != nil {
		return workloads.Workloads{}, nil, fmt.Errorf("failed to discover workloads: %w", err)
	}
	volumes, err := rps.discovery.Volumes(ctx, doguResource.Namespace, doguWorkloads)
	if err != nil {
		return workloads.Workloads{}, nil, fmt.Errorf("failed to discover volumes: %w", err)
	}

	return doguWorkloads, volumes, nil
}

// foreignOwnedClaims finds claims that could be deleted during the resize process. Only StatefulSets are deleted with
// OrphanPropagation, which keeps their owned claims safe. Other unexpectedly owned claims have to be protected and get
// reported by this function.
func foreignOwnedClaims(doguWorkloads workloads.Workloads, volumes []workloads.Volume, claims []string) (failedClaims, foreignOwners []string) {
	for _, volume := range volumes {
		if !slices.Contains(claims, volume.PVC.Name) {
			continue
		}
		var allowed []types.UID
		for _, ref := range volume.MountedBy {
			if workload, _ := findWorkload(doguWorkloads, ref); ref.Kind == "StatefulSet" && workload != nil {
				allowed = append(allowed, workload.GetUID())
			}
		}
		for _, owner := range volume.PVC.OwnerReferences {
			if slices.Contains(allowed, owner.UID) {
				continue
			}
			if !slices.Contains(failedClaims, volume.PVC.Name) {
				failedClaims = append(failedClaims, volume.PVC.Name)
			}
			foreignOwners = append(foreignOwners, fmt.Sprintf("%s (%s/%s)", volume.PVC.Name, owner.Kind, owner.Name))
		}
	}

	return failedClaims, foreignOwners
}

// deleteWorkloadsToRecreate also deletes outdated StatefulSets, as Helm cannot upgrade their immutable claim templates.
func (rps *ResizePVCStep) deleteWorkloadsToRecreate(ctx context.Context, doguWorkloads workloads.Workloads, volumes []workloads.Volume, claims []string, outdatedStatefulSets []types.UID) ([]string, error) {
	var refs []corev1.TypedLocalObjectReference
	for _, volume := range volumes {
		if slices.Contains(claims, volume.PVC.Name) {
			refs = append(refs, volume.MountedBy...)
		}
	}
	// matched by UID, as the checker reads cached and may still see a StatefulSet that Helm has recreated meanwhile
	for _, statefulSet := range doguWorkloads.StatefulSets {
		if slices.Contains(outdatedStatefulSets, statefulSet.UID) {
			refs = append(refs, corev1.TypedLocalObjectReference{Kind: "StatefulSet", Name: statefulSet.Name})
		}
	}

	var deleted []string
	for _, ref := range refs {
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
func (rps *ResizePVCStep) deleteMountingPods(ctx context.Context, namespace string, claims []string) stepsv3.StepResult {
	pods := &corev1.PodList{}
	if err := rps.apiReader.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to list pods in namespace %q: %w", namespace, err), v3beta1.ReasonResizingPVC)
	}

	var deleted []string
	for i := range pods.Items {
		pod := &pods.Items[i]
		// skip finished pods to keep the logs
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed || !podMountsAny(pod, claims) {
			continue
		}
		if err := rps.k8sClient.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
			return stepsv3.RequeueWithError(fmt.Errorf("failed to delete pod %q: %w", pod.Name, err), v3beta1.ReasonResizingPVC)
		}
		deleted = append(deleted, pod.Name)
	}
	if len(deleted) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for pods that mount resizing volumes to terminate: %v", deleted))
	}

	return stepsv3.Continue()
}

func podMountsAny(pod *corev1.Pod, claims []string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && slices.Contains(claims, volume.PersistentVolumeClaim.ClaimName) {
			return true
		}
	}

	return false
}
