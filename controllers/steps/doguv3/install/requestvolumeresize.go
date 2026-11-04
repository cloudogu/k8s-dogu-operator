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

// RequestVolumeResizeStep carries out the pvc.ResizePlan of a dogu, one phase after another, each finished before the next starts:
// it grows the claims, replaces StatefulSets with outdated claim templates and, with offline expansion, unmounts the growing claims.
// Kubernetes grows the volumes afterwards; WaitForVolumeResizeCompletionStep waits for that.
type RequestVolumeResizeStep struct {
	resizePlanner *VolumeResizePlanner
	discovery     VolumeDiscovery
	remover       *workloads.Remover
	grower        *pvc.Grower
	eventRecorder EventRecorder
}

// NewRequestVolumeResizeStep discovers workloads and pods through apiReader, so a lagging cache cannot hide a mounting workload or pod.
func NewRequestVolumeResizeStep(resizePlanner *VolumeResizePlanner, apiReader client.Reader, k8sClient K8sClient, recorder EventRecorder) *RequestVolumeResizeStep {
	return &RequestVolumeResizeStep{
		resizePlanner: resizePlanner,
		discovery:     workloads.NewDiscovery(apiReader),
		remover:       workloads.NewRemover(apiReader, k8sClient),
		grower:        pvc.NewGrower(k8sClient),
		eventRecorder: recorder,
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
	if !plan.HasWork() {
		return stepsv3.Continue()
	}

	doguWorkloads, volumes, err := rvr.discover(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	// abort before any change, as unmounting must not delete a claim
	if foreignOwners := workloads.ForeignOwnedClaims(doguWorkloads, volumes, plan.Unmount); len(foreignOwners) > 0 {
		return stepsv3.Abort(v3beta1.ReasonResizingPVC, fmt.Sprintf("cannot resize volumes owned by other objects than their StatefulSet, their deletion would lead to data loss: %v", foreignOwners))
	}
	if result := rvr.growClaims(ctx, doguResource, plan.Grow); !result.Continue {
		return result
	}
	if result := rvr.replaceStatefulSets(ctx, doguResource, doguWorkloads, plan.OutdatedStatefulSets); !result.Continue {
		return result
	}
	return rvr.unmount(ctx, doguResource, doguWorkloads, volumes, plan.Unmount)
}

// growClaims grows the volume claims. It aborts if the cluster rejects a claim, before any workload is deleted.
func (rvr *RequestVolumeResizeStep) growClaims(ctx context.Context, doguResource *v3beta1.Dogu, requests []pvc.ResizeRequest) stepsv3.StepResult {
	if len(requests) == 0 {
		return stepsv3.Continue()
	}

	grown, err := rvr.grower.Grow(ctx, requests)
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

// replaceStatefulSets deletes the outdated StatefulSets and waits until they are gone.
func (rvr *RequestVolumeResizeStep) replaceStatefulSets(ctx context.Context, doguResource *v3beta1.Dogu, doguWorkloads workloads.Workloads, outdated []types.UID) stepsv3.StepResult {
	deleted, err := rvr.remover.ReplaceStatefulSets(ctx, doguWorkloads, outdated)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}
	if len(deleted) == 0 {
		return stepsv3.Continue()
	}

	logging.Info(ctx, "deleted StatefulSets with outdated claim templates", "workloads", deleted)
	rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted %v with outdated claim templates, the release recreates them", deleted))
	return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for StatefulSets with outdated claim templates to be deleted: %v", deleted))
}

// unmount deletes the workloads and pods mounting the claims and waits until they are gone.
func (rvr *RequestVolumeResizeStep) unmount(ctx context.Context, doguResource *v3beta1.Dogu, doguWorkloads workloads.Workloads, volumes []pvc.Volume, claims []string) stepsv3.StepResult {
	unmounted, err := rvr.remover.Unmount(ctx, doguResource.Namespace, doguWorkloads, volumes, claims)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonResizingPVC)
	}

	if len(unmounted.Workloads) > 0 {
		logging.Info(ctx, "deleted workloads that mount resizing volumes", "workloads", unmounted.Workloads)
		rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted %v that mount resizing volumes", unmounted.Workloads))
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for workloads that mount resizing volumes to be deleted: %v", unmounted.Workloads))
	}
	if len(unmounted.NewlyDeletedPods) > 0 {
		logging.Info(ctx, "deleted pods that mount resizing volumes", "pods", unmounted.NewlyDeletedPods)
		rvr.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonResizingPVC, fmt.Sprintf("Deleted pods %v that mount resizing volumes", unmounted.NewlyDeletedPods))
	}
	if len(unmounted.Pods) > 0 {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonResizingPVC, fmt.Sprintf("waiting for pods that mount resizing volumes to terminate: %v", unmounted.Pods))
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
