package workloads

import (
	"context"
	"fmt"

	flux "github.com/fluxcd/helm-controller/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ScaleState tells how far scaling down the workloads has progressed.
type ScaleState int

const (
	// ScaledDown means all workloads have zero replicas and none of their Pods remain.
	ScaledDown ScaleState = iota
	// ScalingDown means zero replicas were requested, but workloads or their Pods are still being removed.
	ScalingDown
	// WaitingForSuspension means the HelmRelease is not suspended yet. No workload was touched.
	WaitingForSuspension
	// WaitingForRelease means workloads exist without a HelmRelease that could be suspended. No workload was touched.
	WaitingForRelease
)

// Scaler scales Deployments and StatefulSets of a HelmRelease to zero without losing their volumes.
type Scaler struct {
	k8s client.Client
}

func NewScaler(k8s client.Client) *Scaler {
	return &Scaler{k8s: k8s}
}

// ScaleDown scales the given Deployments and StatefulSets to zero, once the HelmRelease is suspended.
func (s *Scaler) ScaleDown(ctx context.Context, releaseKey client.ObjectKey, workloads Workloads) (ScaleState, error) {
	release := &flux.HelmRelease{}
	err := s.k8s.Get(ctx, releaseKey, release)
	releaseMissing := apierrors.IsNotFound(err)
	if err != nil && !releaseMissing {
		return ScalingDown, err
	}

	// Flux must stop applying the release before we scale its workloads down.
	if !releaseMissing && (!release.Spec.Suspend || release.Status.ObservedGeneration != release.Generation) {
		return WaitingForSuspension, nil
	}

	// Without a release there is nobody to suspend. Do not mutate orphaned workloads.
	if releaseMissing && !workloads.IsEmpty() {
		return WaitingForRelease, nil
	}

	waiting, err := s.scaleWorkloads(ctx, workloads)
	if err != nil {
		return ScalingDown, err
	}

	if waiting {
		return ScalingDown, nil
	}

	// Zero replicas are not enough; wait until the associated Pods are gone.
	podsRemain, err := s.podsRemain(ctx, releaseKey.Namespace, workloads)
	if err != nil {
		return ScalingDown, err
	}

	if podsRemain {
		return ScalingDown, nil
	}

	return ScaledDown, nil
}

// scaleWorkloads requests zero replicas. waiting means: call again.
func (s *Scaler) scaleWorkloads(ctx context.Context, workloads Workloads) (bool, error) {
	waiting := false

	for i := range workloads.Deployments {
		deployment := &workloads.Deployments[i]
		if !ptr.Equal(deployment.Spec.Replicas, ptr.To[int32](0)) {
			before := deployment.DeepCopy()
			deployment.Spec.Replicas = ptr.To[int32](0)
			if err := s.k8s.Patch(ctx, deployment, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
				return false, err
			}

			waiting = true
		}
	}

	for i := range workloads.StatefulSets {
		statefulSet := &workloads.StatefulSets[i]
		workloadWaiting, err := s.scaleStatefulSet(ctx, statefulSet)
		if err != nil {
			return false, err
		}

		waiting = waiting || workloadWaiting
	}

	return waiting, nil
}

// scaleStatefulSet scales to zero without losing PVCs, one step per call. waiting means: call again.
func (s *Scaler) scaleStatefulSet(ctx context.Context, statefulSet *appsv1.StatefulSet) (waiting bool, err error) {
	// Retain PVCs on scale-down without changing what happens when the StatefulSet is deleted.
	if policy := statefulSet.Spec.PersistentVolumeClaimRetentionPolicy; policy != nil && policy.WhenScaled == appsv1.DeletePersistentVolumeClaimRetentionPolicyType {
		before := statefulSet.DeepCopy()
		statefulSet.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled = appsv1.RetainPersistentVolumeClaimRetentionPolicyType
		if err = s.k8s.Patch(ctx, statefulSet, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
			return false, fmt.Errorf("failed to retain PVCs when scaling StatefulSet %s/%s: %w", statefulSet.Namespace, statefulSet.Name, err)
		}

		// Wait for the StatefulSet controller to remove Pod owner references from PVCs before scaling down.
		return true, nil
	}

	if ptr.Equal(statefulSet.Spec.Replicas, ptr.To[int32](0)) {
		return false, nil
	}

	// Check that PVCs are safe before reducing the replica count.
	safe, err := s.claimsRetained(ctx, statefulSet)
	if err != nil {
		return false, err
	}

	if !safe {
		return true, nil
	}

	before := statefulSet.DeepCopy()
	statefulSet.Spec.Replicas = ptr.To[int32](0)
	if err = s.k8s.Patch(ctx, statefulSet, client.MergeFromWithOptions(before, client.MergeFromWithOptimisticLock{})); err != nil {
		return false, fmt.Errorf("failed to scale StatefulSet %s/%s to zero: %w", statefulSet.Namespace, statefulSet.Name, err)
	}

	return true, nil
}

// claimsRetained checks that the StatefulSet's PVCs are not being deleted and no longer have Pod owner references.
// PVCs are identified by name, even without a dogu label.
func (s *Scaler) claimsRetained(ctx context.Context, statefulSet *appsv1.StatefulSet) (bool, error) {
	claims := &corev1.PersistentVolumeClaimList{}
	if err := s.k8s.List(ctx, claims, client.InNamespace(statefulSet.Namespace)); err != nil {
		return false, err
	}

	for _, claim := range claims.Items {
		if !hasClaimTemplateFor(*statefulSet, claim.Name) {
			continue
		}

		if !claim.DeletionTimestamp.IsZero() {
			return false, fmt.Errorf("PVC %s/%s is already being deleted", claim.Namespace, claim.Name)
		}

		// Wait until the StatefulSet controller removes ownership by Pods that will be deleted.
		for _, owner := range claim.OwnerReferences {
			if owner.APIVersion == "v1" && owner.Kind == "Pod" {
				return false, nil
			}
		}
	}

	return true, nil
}

// podsRemain reports whether Pods of the workloads still exist, including terminating ones.
func (s *Scaler) podsRemain(ctx context.Context, namespace string, workloads Workloads) (bool, error) {
	replicaSets := &appsv1.ReplicaSetList{}
	if err := s.k8s.List(ctx, replicaSets, client.InNamespace(namespace)); err != nil {
		return false, err
	}

	pods := &corev1.PodList{}
	if err := s.k8s.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return false, err
	}

	selectors, err := deploymentSelectors(workloads.Deployments)
	if err != nil {
		return false, err
	}

	// StatefulSets own their Pods directly, Deployments through their ReplicaSets.
	owned := workloadUIDs(workloads)
	existingReplicaSets := make(map[types.UID]bool, len(replicaSets.Items))
	for _, replicaSet := range replicaSets.Items {
		existingReplicaSets[replicaSet.UID] = true
		if owner := metav1.GetControllerOf(&replicaSet); owner != nil && owned[owner.UID] {
			owned[replicaSet.UID] = true
		}
	}

	for _, pod := range pods.Items {
		owner := metav1.GetControllerOf(&pod)
		switch {
		case owner == nil:
			continue
		case owned[owner.UID]:
			return true, nil
		// A terminating Pod can outlive its ReplicaSet; then only the Deployment selector identifies it.
		case owner.Kind == "ReplicaSet" && !existingReplicaSets[owner.UID] && matchesAny(selectors, pod.Labels):
			return true, nil
		}
	}

	return false, nil
}

func workloadUIDs(workloads Workloads) map[types.UID]bool {
	uids := make(map[types.UID]bool, len(workloads.Deployments)+len(workloads.StatefulSets))
	for _, deployment := range workloads.Deployments {
		uids[deployment.UID] = true
	}

	for _, statefulSet := range workloads.StatefulSets {
		uids[statefulSet.UID] = true
	}

	return uids
}

func deploymentSelectors(deployments []appsv1.Deployment) ([]labels.Selector, error) {
	selectors := make([]labels.Selector, 0, len(deployments))
	for _, deployment := range deployments {
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if err != nil {
			return nil, err
		}

		selectors = append(selectors, selector)
	}

	return selectors, nil
}

func matchesAny(selectors []labels.Selector, podLabels map[string]string) bool {
	for _, selector := range selectors {
		if selector.Matches(labels.Set(podLabels)) {
			return true
		}
	}

	return false
}
