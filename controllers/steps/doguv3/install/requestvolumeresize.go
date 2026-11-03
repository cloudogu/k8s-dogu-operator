package install

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RequestVolumeResizeStep requests the pvc.ResizePlan of a dogu: it raises the PVC requests and replaces StatefulSets with
// outdated claim templates. With offline expansion it also detaches every growing PVC, so that the resumed release recreates
// the workloads on them. Kubernetes grows the volumes afterwards; WaitForVolumeResizeCompletionStep waits for that.
type RequestVolumeResizeStep struct {
	resizePlanner    *VolumeResizePlanner
	discovery        VolumeDiscovery
	detacher         *workloads.Detacher
	grower           *pvc.Grower
	eventRecorder    EventRecorder
	offlineExpansion bool
}

// NewRequestVolumeResizeStep discovers and detaches through apiReader, so a lagging cache cannot hide a mounting workload or pod.
func NewRequestVolumeResizeStep(resizePlanner *VolumeResizePlanner, apiReader client.Reader, k8sClient K8sClient, recorder EventRecorder, offlineExpansion bool) *RequestVolumeResizeStep {
	return &RequestVolumeResizeStep{
		resizePlanner:    resizePlanner,
		discovery:        workloads.NewDiscovery(apiReader),
		detacher:         workloads.NewDetacher(apiReader, k8sClient),
		grower:           pvc.NewGrower(k8sClient),
		eventRecorder:    recorder,
		offlineExpansion: offlineExpansion,
	}
}

func (rvr *RequestVolumeResizeStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}

	plan, err := rvr.resizePlanner.Plan(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to plan resizing of pvcs: %w", err), v3beta1.ReasonResizingPVC)
	}

	if rvr.offlineExpansion {
		return rvr.resizeOffline(ctx, doguResource, plan)
	}
	return rvr.resizeOnline(ctx, doguResource, plan)
}

// resizeOnline grows the claims while their workloads and pods keep running.
func (rvr *RequestVolumeResizeStep) resizeOnline(ctx context.Context, doguResource *v3beta1.Dogu, plan pvc.ResizePlan) stepsv3.StepResult {
	if len(plan.Grow) == 0 && len(plan.OutdatedStatefulSets) == 0 {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, err := rvr.discover(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	if result := rvr.growClaims(ctx, doguResource, plan); !result.Continue {
		return result
	}

	// mounted volumes grow online, only StatefulSets with outdated claim templates are replaced
	return rvr.deleteWorkloads(ctx, doguResource, doguWorkloads, volumes, nil, plan.OutdatedStatefulSets)
}

// resizeOffline detaches the claims from all workloads and pods, for storage that only expands unmounted volumes.
func (rvr *RequestVolumeResizeStep) resizeOffline(ctx context.Context, doguResource *v3beta1.Dogu, plan pvc.ResizePlan) stepsv3.StepResult {
	if !plan.Active() {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, err := rvr.discover(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	failedClaims, foreignOwners := workloads.ForeignOwnedClaims(doguWorkloads, volumes, plan.Detach)
	if len(failedClaims) > 0 {
		return stepsv3.Abort(v3beta1.ReasonResizingPVC, fmt.Sprintf("cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: %v", foreignOwners))
	}

	if result := rvr.growClaims(ctx, doguResource, plan); !result.Continue {
		return result
	}
	if result := rvr.deleteWorkloads(ctx, doguResource, doguWorkloads, volumes, plan.Detach, plan.OutdatedStatefulSets); !result.Continue {
		return result
	}
	return rvr.deleteMountingPods(ctx, doguResource, plan.Detach)
}

// deleteWorkloads deletes the workloads mounting the claims plus the outdated StatefulSets and waits for the release to recreate them.
func (rvr *RequestVolumeResizeStep) deleteWorkloads(ctx context.Context, doguResource *v3beta1.Dogu, doguWorkloads workloads.Workloads, volumes []pvc.Volume, claims []string, outdatedStatefulSets []types.UID) stepsv3.StepResult {
	deleted, err := rvr.detacher.DeleteWorkloads(ctx, doguWorkloads, volumes, claims, outdatedStatefulSets)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deleted) == 0 {
		return stepsv3.Continue()
	}

	logging.Info(ctx, "deleted workloads to recreate them on the resized volumes", "workloads", deleted)
	rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted %v to recreate them on the resized volumes", deleted))
	return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for workloads to be deleted and recreated by the release: %v", deleted))
}

// deleteMountingPods deletes the pods mounting the claims and waits until they are gone.
func (rvr *RequestVolumeResizeStep) deleteMountingPods(ctx context.Context, doguResource *v3beta1.Dogu, claims []string) stepsv3.StepResult {
	pods, newlyDeleted, err := rvr.detacher.DeleteMountingPods(ctx, doguResource.Namespace, claims)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(newlyDeleted) > 0 {
		logging.Info(ctx, "deleted pods that mount resizing volumes", "pods", newlyDeleted)
		rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted pods %v that mount resizing volumes", newlyDeleted))
	}
	if len(pods) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for pods that mount resizing volumes to terminate: %v", pods))
	}

	return stepsv3.Continue()
}

// growClaims grows the volume claims. It aborts if the cluster rejects a claim, before any workload is deleted.
func (rvr *RequestVolumeResizeStep) growClaims(ctx context.Context, doguResource *v3beta1.Dogu, plan pvc.ResizePlan) stepsv3.StepResult {
	if len(plan.Grow) == 0 {
		return stepsv3.Continue()
	}

	grown, err := rvr.grower.Grow(ctx, plan.Grow)
	for _, claim := range grown {
		from, to := claim.Previous.String(), claim.PVC.Spec.Resources.Requests.Storage().String()
		logging.Info(ctx, "growing volume", "pvc", claim.PVC.Name, "from", from, "to", to)
		rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Growing volume %s from %s to %s", claim.PVC.Name, from, to))
	}

	var rejection *pvc.GrowRejectedError
	if err != nil && !errors.As(err, &rejection) {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if rejection != nil {
		return stepsv3.Abort(v3beta1.ReasonResizingPVC, rejection.Error())
	}

	return stepsv3.Continue()
}

func (rvr *RequestVolumeResizeStep) discover(ctx context.Context, doguResource *v3beta1.Dogu) (workloads.Workloads, []pvc.Volume, error) {
	doguWorkloads, err := rvr.discovery.Workloads(ctx, doguResource.Namespace, doguResource.Spec.Name)
	if err != nil {
		return workloads.Workloads{}, nil, fmt.Errorf("failed to discover workloads: %w", err)
	}
	volumes, err := rvr.discovery.Volumes(ctx, doguResource.Namespace, doguWorkloads)
	if err != nil {
		return workloads.Workloads{}, nil, fmt.Errorf("failed to discover volumes: %w", err)
	}

	return doguWorkloads, volumes, nil
}
