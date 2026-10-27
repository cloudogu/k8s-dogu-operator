package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizePVCStep carries out the VolumeResizePlan of a dogu: it grows the PVCs and replaces StatefulSets with outdated
// claim templates. With offline expansion it also detaches every growing PVC, so that the resumed release recreates
// the workloads on the grown volumes.
type ResizePVCStep struct {
	resizePlanner    *VolumeResizePlanner
	discovery        workloads.Discovery
	detacher         *workloads.Detacher
	k8sClient        K8sClient
	eventRecorder    EventRecorder
	offlineExpansion bool
}

// NewResizePVCStep discovers and detaches through apiReader, so a lagging cache cannot hide a mounting workload or pod.
func NewResizePVCStep(resizePlanner *VolumeResizePlanner, apiReader client.Reader, k8sClient K8sClient, recorder EventRecorder, offlineExpansion bool) *ResizePVCStep {
	return &ResizePVCStep{
		resizePlanner:    resizePlanner,
		discovery:        workloads.NewDiscovery(apiReader),
		detacher:         workloads.NewDetacher(apiReader, k8sClient),
		k8sClient:        k8sClient,
		eventRecorder:    recorder,
		offlineExpansion: offlineExpansion,
	}
}

func (rps *ResizePVCStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	plan, err := rps.resizePlanner.Plan(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	if rps.offlineExpansion {
		return rps.resizeOffline(ctx, doguResource, plan)
	}
	return rps.resizeOnline(ctx, doguResource, plan)
}

// resizeOnline grows the claims while their workloads and pods keep running.
func (rps *ResizePVCStep) resizeOnline(ctx context.Context, doguResource *v3beta1.Dogu, plan VolumeResizePlan) stepsv3.StepResult {
	if len(plan.Grow) == 0 && len(plan.OutdatedStatefulSets) == 0 {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, result := rps.growClaims(ctx, doguResource, plan)
	if !result.Continue {
		return result
	}

	// mounted volumes grow online, only StatefulSets with outdated claim templates are replaced
	return rps.deleteWorkloads(ctx, doguResource, doguWorkloads, volumes, nil, plan.OutdatedStatefulSets)
}

// resizeOffline detaches the claims from all workloads and pods, for storage that only expands unmounted volumes.
func (rps *ResizePVCStep) resizeOffline(ctx context.Context, doguResource *v3beta1.Dogu, plan VolumeResizePlan) stepsv3.StepResult {
	if !plan.Active() {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, err := rps.discover(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	failedClaims, foreignOwners := workloads.ForeignOwnedClaims(doguWorkloads, volumes, plan.Detach)
	if len(failedClaims) > 0 {
		if _, err := updateVolumeStatus(ctx, rps.k8sClient, doguResource, plan.rendered, volumes, failedClaims); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
		}
		return stepsv3.Abort(v3beta1.ReasonResizingPVC, fmt.Sprintf("cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: %v", foreignOwners))
	}

	var result stepsv3.StepResult
	doguWorkloads, volumes, result = rps.growClaims(ctx, doguResource, plan)
	if !result.Continue {
		return result
	}

	if result := rps.deleteWorkloads(ctx, doguResource, doguWorkloads, volumes, plan.Detach, plan.OutdatedStatefulSets); !result.Continue {
		return result
	}
	return rps.deleteMountingPods(ctx, doguResource, plan.Detach)
}

// deleteWorkloads deletes the workloads mounting the claims plus the outdated StatefulSets and waits for the release to recreate them.
func (rps *ResizePVCStep) deleteWorkloads(ctx context.Context, doguResource *v3beta1.Dogu, doguWorkloads workloads.Workloads, volumes []workloads.Volume, claims []string, outdatedStatefulSets []types.UID) stepsv3.StepResult {
	deleted, err := rps.detacher.DeleteWorkloads(ctx, doguWorkloads, volumes, claims, outdatedStatefulSets)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deleted) == 0 {
		return stepsv3.Continue()
	}

	logging.Info(ctx, "deleted workloads to recreate them on the resized volumes", "workloads", deleted)
	rps.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted %v to recreate them on the resized volumes", deleted))
	return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for workloads to be deleted and recreated by the release: %v", deleted))
}

// deleteMountingPods deletes the pods mounting the claims and waits until they are gone.
func (rps *ResizePVCStep) deleteMountingPods(ctx context.Context, doguResource *v3beta1.Dogu, claims []string) stepsv3.StepResult {
	pods, newlyDeleted, err := rps.detacher.DeleteMountingPods(ctx, doguResource.Namespace, claims)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(newlyDeleted) > 0 {
		logging.Info(ctx, "deleted pods that mount resizing volumes", "pods", newlyDeleted)
		rps.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted pods %v that mount resizing volumes", newlyDeleted))
	}
	if len(pods) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for pods that mount resizing volumes to terminate: %v", pods))
	}

	return stepsv3.Continue()
}

// growClaims grows and reports the volume claims. It aborts if the cluster rejects a claim, before any workload is deleted.
func (rps *ResizePVCStep) growClaims(ctx context.Context, doguResource *v3beta1.Dogu, plan VolumeResizePlan) (workloads.Workloads, []workloads.Volume, stepsv3.StepResult) {
	var rejectedClaims []string
	var rejection error
	for _, request := range plan.Grow {
		err := rps.patchStorageRequest(ctx, doguResource, request)
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
	// without new requests the workloads may be deleted, so the discovered volumes would wipe the status of a running resize
	if len(plan.Grow) > 0 {
		if _, err := updateVolumeStatus(ctx, rps.k8sClient, doguResource, plan.rendered, volumes, rejectedClaims); err != nil {
			return workloads.Workloads{}, nil, stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
		}
	}
	if rejection != nil {
		return workloads.Workloads{}, nil, stepsv3.Abort(v3beta1.ReasonResizingPVC, rejection.Error())
	}

	return doguWorkloads, volumes, stepsv3.Continue()
}

func (rps *ResizePVCStep) patchStorageRequest(ctx context.Context, doguResource *v3beta1.Dogu, request pvc.ResizeRequest) error {
	claim := &corev1.PersistentVolumeClaim{}
	if err := rps.k8sClient.Get(ctx, request.PVC, claim); err != nil {
		return fmt.Errorf("failed to get PVC %q: %w", request.PVC, err)
	}

	current := claim.Spec.Resources.Requests.Storage().String()
	patch := client.MergeFrom(claim.DeepCopy())
	claim.Spec.Resources.Requests[corev1.ResourceStorage] = request.Desired.DeepCopy()
	if err := rps.k8sClient.Patch(ctx, claim, patch); err != nil {
		return fmt.Errorf("failed to grow PVC %q to %s: %w", request.PVC, request.Desired.String(), err)
	}

	logging.Info(ctx, "growing volume", "pvc", claim.Name, "from", current, "to", request.Desired.String())
	rps.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Growing volume %s from %s to %s", claim.Name, current, request.Desired.String()))
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
