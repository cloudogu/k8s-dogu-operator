package install

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	flux "github.com/fluxcd/helm-controller/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StopWorkloadsStep struct {
	k8sClient client.Client
	discovery workloads.Discovery
}

func NewStopWorkloadsStep(k8sClient client.Client) *StopWorkloadsStep {
	return &StopWorkloadsStep{k8sClient: k8sClient, discovery: workloads.NewDiscovery(k8sClient)}
}

func (s *StopWorkloadsStep) Run(ctx context.Context, dogu *v3beta1.Dogu) stepsv3.StepResult {
	if dogu.Spec.PauseReconciliation {
		return stepsv3.Abort(v3beta1.ReasonPauseReconciliation, "Dogu reconciliation is paused")
	}
	if !dogu.Spec.Stopped {
		return stepsv3.Continue()
	}

	release := &flux.HelmRelease{}
	err := s.k8sClient.Get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, release)
	releaseMissing := apierrors.IsNotFound(err)
	if err != nil && !releaseMissing {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
	}
	// Record a known version change even when the new chart cannot be fetched.
	pending := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	if releaseMissing || release.Labels[v3beta1.DoguLabelVersion] != dogu.Spec.Version {
		if err := updatePending(ctx, s.k8sClient, dogu, metav1.ConditionTrue, "A chart version or installation is pending until the dogu starts"); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
	} else if pending == nil || pending.ObservedGeneration != dogu.Generation {
		if err := updatePending(ctx, s.k8sClient, dogu, metav1.ConditionUnknown, "The desired release has not been compared yet"); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
	}
	if !releaseMissing && !release.Spec.Suspend {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for HelmRelease suspension before stopping workloads")
	}
	// Suspension does not cancel an action already in flight. Flux acknowledges
	// the suspended generation only after that reconciliation has returned.
	if !releaseMissing && release.Status.ObservedGeneration != release.Generation {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for Flux to acknowledge the suspended HelmRelease generation")
	}

	found, err := s.discovery.Workloads(ctx, dogu.Namespace, dogu.Spec.Name)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
	}
	// Without a release there is nobody to suspend. Do not mutate orphaned workloads.
	if releaseMissing && !found.IsEmpty() {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for a suspended HelmRelease for existing workloads")
	}

	owners := make(map[types.UID]bool)
	waiting := false
	// scale deployments
	for i := range found.Deployments {
		deployment := &found.Deployments[i]
		owners[deployment.UID] = true
		if !ptr.Equal(deployment.Spec.Replicas, ptr.To[int32](0)) {
			before := deployment.DeepCopy()
			deployment.Spec.Replicas = ptr.To[int32](0)
			if err := s.k8sClient.Patch(ctx, deployment, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
				return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
			}
			waiting = true
		}
	}
	// scale statefulsets
	for i := range found.StatefulSets {
		statefulSet := &found.StatefulSets[i]
		owners[statefulSet.UID] = true
		workloadWaiting, err := s.stopStatefulSet(ctx, statefulSet)
		if err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
		waiting = waiting || workloadWaiting
	}
	if len(owners) > 0 {
		podsRemain, err := s.podsRemain(ctx, dogu.Namespace, owners, found.Deployments)
		if err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
		waiting = waiting || podsRemain
	}

	// Zero desired replicas alone do not confirm a stop; the associated Pods must also be gone.
	status, reason, message := metav1.ConditionTrue, v3beta1.ReasonSucceeded, "All dogu workloads are stopped"
	if waiting {
		status, reason, message = metav1.ConditionFalse, v3beta1.ReasonPartiallyStopped, "Waiting for all dogu workloads and Pods to stop"
	}
	stopped := !waiting
	stoppedChanged := dogu.Status.Stopped != stopped
	dogu.Status.Stopped = stopped

	conditionChanged := meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionStopped,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: dogu.Generation,
	})
	if stoppedChanged || conditionChanged {
		if err := s.k8sClient.Status().Update(ctx, dogu); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
	}
	if waiting {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, message)
	}
	return stepsv3.Continue()
}

// stopStatefulSet protects claims before scaling. A true result requires another
// reconciliation; Pod termination is checked separately by the caller.
func (s *StopWorkloadsStep) stopStatefulSet(ctx context.Context, statefulSet *appsv1.StatefulSet) (waiting bool, err error) {
	if policy := statefulSet.Spec.PersistentVolumeClaimRetentionPolicy; policy != nil && policy.WhenScaled == appsv1.DeletePersistentVolumeClaimRetentionPolicyType {
		before := statefulSet.DeepCopy()
		statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled = appsv1.RetainPersistentVolumeClaimRetentionPolicyType
		if err := s.k8sClient.Patch(ctx, statefulSet, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return false, fmt.Errorf("failed to retain PVCs when scaling StatefulSet %s/%s: %w", statefulSet.Namespace, statefulSet.Name, err)
		}
		return true, nil // Wait for the StatefulSet controller to remove Pod owner references from PVCs before scaling down.
	}
	if ptr.Equal(statefulSet.Spec.Replicas, ptr.To[int32](0)) {
		return false, nil
	}
	safe, err := s.claimsRetained(ctx, statefulSet)
	if err != nil {
		return false, err
	}
	if !safe {
		return true, nil
	}
	before := statefulSet.DeepCopy()
	statefulSet.Spec.Replicas = ptr.To[int32](0)
	if err := s.k8sClient.Patch(ctx, statefulSet, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
		return false, fmt.Errorf("failed to scale StatefulSet %s/%s to zero: %w", statefulSet.Namespace, statefulSet.Name, err)
	}
	return true, nil
}

// claimsRetained checks that the StatefulSet's PVCs are not being deleted and no longer
// have Pod owner references. PVCs are identified by name, even without a dogu label.
func (s *StopWorkloadsStep) claimsRetained(ctx context.Context, statefulSet *appsv1.StatefulSet) (bool, error) {
	claims := &corev1.PersistentVolumeClaimList{}
	if err := s.k8sClient.List(ctx, claims, client.InNamespace(statefulSet.Namespace)); err != nil {
		return false, err
	}
	for _, claim := range claims.Items {
		for _, template := range statefulSet.Spec.VolumeClaimTemplates {
			ordinal, found := strings.CutPrefix(claim.Name, template.Name+"-"+statefulSet.Name+"-")
			if !found {
				continue
			}
			if n, err := strconv.ParseUint(ordinal, 10, 32); err != nil || fmt.Sprint(n) != ordinal {
				continue
			}
			if !claim.DeletionTimestamp.IsZero() {
				return false, fmt.Errorf("PVC %s/%s is already being deleted", claim.Namespace, claim.Name)
			}
			for _, owner := range claim.OwnerReferences {
				if owner.APIVersion == "v1" && owner.Kind == "Pod" {
					return false, nil
				}
			}
		}
	}
	return true, nil
}

// podsRemain checks whether any Pods still belong to the workloads, including terminating Pods.
// Deployment Pods are identified through their ReplicaSets; StatefulSet Pods are owned directly.
func (s *StopWorkloadsStep) podsRemain(ctx context.Context, namespace string, owners map[types.UID]bool, deployments []appsv1.Deployment) (bool, error) {
	replicaSets := &appsv1.ReplicaSetList{}
	if err := s.k8sClient.List(ctx, replicaSets, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	knownReplicaSets := make(map[types.UID]bool, len(replicaSets.Items))
	for _, replicaSet := range replicaSets.Items {
		knownReplicaSets[replicaSet.UID] = true
		if owner := metav1.GetControllerOf(&replicaSet); owner != nil && owners[owner.UID] {
			owners[replicaSet.UID] = true
		}
	}
	selectors := make([]labels.Selector, 0, len(deployments))
	for _, deployment := range deployments {
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			return false, err
		}
		selectors = append(selectors, selector)
	}
	pods := &corev1.PodList{}
	if err := s.k8sClient.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return false, err
	}
	for _, pod := range pods.Items {
		owner := metav1.GetControllerOf(&pod)
		if owner == nil {
			continue
		}
		if owners[owner.UID] {
			return true, nil
		}
		// Keep waiting for terminating Pods even if their ReplicaSet has already been deleted.
		// In that case, use Deployment selectors to identify them.
		// Never use matching labels to override an existing ReplicaSet owned by another Deployment.
		if owner.APIVersion == "apps/v1" && owner.Kind == "ReplicaSet" && !knownReplicaSets[owner.UID] {
			for _, selector := range selectors {
				if selector.Matches(labels.Set(pod.Labels)) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
