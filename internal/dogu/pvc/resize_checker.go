package pvc

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizeRequest describes either a required PVC expansion or a PVC-specific validation error.
type ResizeRequest struct {
	PVC     client.ObjectKey
	Desired resource.Quantity
	Err     error
}

// CheckResult contains only PVCs that require an expansion or have a validation error.
type CheckResult struct {
	ResizeRequests []ResizeRequest
}

// ResizeChecker checks rendered Kubernetes objects against live PVCs without changing cluster resources.
type ResizeChecker interface {
	Check(ctx context.Context, renderedObjects []*unstructured.Unstructured) (CheckResult, error)
}

type resizeChecker struct {
	k8s client.Reader
}

// NewResizeChecker creates a checker that reads live PVCs with the given Kubernetes client.
func NewResizeChecker(k8s client.Reader) ResizeChecker {
	return &resizeChecker{k8s: k8s}
}

type desiredClaim struct {
	key          client.ObjectKey
	storage      resource.Quantity
	storageClass *string
}

func (c *resizeChecker) Check(ctx context.Context, renderedObjects []*unstructured.Unstructured) (CheckResult, error) {
	desiredClaims, err := extractDesiredClaims(renderedObjects)
	if err != nil {
		return CheckResult{}, err
	}

	result := CheckResult{}
	for _, desired := range desiredClaims {
		request, include, checkErr := c.checkPVC(ctx, desired)
		if checkErr != nil {
			return CheckResult{}, checkErr
		}
		if include {
			result.ResizeRequests = append(result.ResizeRequests, request)
		}
	}

	return result, nil
}

func (c *resizeChecker) checkPVC(ctx context.Context, desired desiredClaim) (ResizeRequest, bool, error) {
	livePVC := &corev1.PersistentVolumeClaim{}
	err := c.k8s.Get(ctx, desired.key, livePVC)
	if apierrors.IsNotFound(err) {
		return ResizeRequest{}, false, nil
	}
	if err != nil {
		return ResizeRequest{}, false, fmt.Errorf("failed to read live PVC %q: %w", desired.key, err)
	}

	current, found := livePVC.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return ResizeRequest{}, false, fmt.Errorf("live PVC %q has no storage request", desired.key)
	}

	request := ResizeRequest{
		PVC:     desired.key,
		Desired: desired.storage.DeepCopy(),
	}

	if desired.storageClass != nil && !equalStringPointers(desired.storageClass, livePVC.Spec.StorageClassName) {
		request.Err = &StorageClassImmutableError{
			PVC:     desired.key,
			Current: copyStringPointer(livePVC.Spec.StorageClassName),
			Desired: copyStringPointer(desired.storageClass),
		}
	}

	switch desired.storage.Cmp(current) {
	case -1:
		request.Err = errors.Join(request.Err, &VolumeShrinkError{
			PVC:     desired.key,
			Current: current.DeepCopy(),
			Desired: desired.storage.DeepCopy(),
		})
	case 0:
		if request.Err == nil {
			return ResizeRequest{}, false, nil
		}
	}

	return request, true, nil
}

func extractDesiredClaims(renderedObjects []*unstructured.Unstructured) ([]desiredClaim, error) {
	var claims []desiredClaim

	for _, unstructuredObject := range renderedObjects {
		switch unstructuredObject.GetKind() {
		case "PersistentVolumeClaim":
			pvc := &corev1.PersistentVolumeClaim{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, pvc); err != nil {
				return nil, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err)
			}

			claim, err := desiredClaimFromPVC(pvc)
			if err != nil {
				return nil, err
			}
			claims = append(claims, claim)
		case "StatefulSet":
			statefulSet := &appsv1.StatefulSet{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, statefulSet); err != nil {
				return nil, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err)
			}

			statefulSetClaims, err := desiredClaimsFromStatefulSet(statefulSet)
			if err != nil {
				return nil, err
			}
			claims = append(claims, statefulSetClaims...)
		}
	}

	return claims, nil
}

func desiredClaimsFromStatefulSet(statefulSet *appsv1.StatefulSet) ([]desiredClaim, error) {
	if statefulSet.Namespace == "" {
		return nil, fmt.Errorf("rendered StatefulSet %q has no namespace", statefulSet.Name)
	}

	replicas := int32(1)
	if statefulSet.Spec.Replicas != nil {
		replicas = *statefulSet.Spec.Replicas
	}
	if replicas < 0 {
		return nil, fmt.Errorf("rendered StatefulSet %q has negative replicas", client.ObjectKeyFromObject(statefulSet))
	}

	startOrdinal := int32(0)
	if statefulSet.Spec.Ordinals != nil {
		startOrdinal = statefulSet.Spec.Ordinals.Start
	}

	claims := make([]desiredClaim, 0, len(statefulSet.Spec.VolumeClaimTemplates)*int(replicas))
	for _, template := range statefulSet.Spec.VolumeClaimTemplates {
		for ordinal := startOrdinal; ordinal < startOrdinal+replicas; ordinal++ {
			pvc := template.DeepCopy()
			pvc.Name = fmt.Sprintf("%s-%s-%d", template.Name, statefulSet.Name, ordinal)
			pvc.Namespace = statefulSet.Namespace

			claim, err := desiredClaimFromPVC(pvc)
			if err != nil {
				return nil, err
			}
			claims = append(claims, claim)
		}
	}

	return claims, nil
}

func desiredClaimFromPVC(pvc *corev1.PersistentVolumeClaim) (desiredClaim, error) {
	if pvc.Namespace == "" {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q has no namespace", pvc.Name)
	}

	storage, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q has no storage request", client.ObjectKeyFromObject(pvc))
	}

	return desiredClaim{
		key:          client.ObjectKeyFromObject(pvc),
		storage:      storage.DeepCopy(),
		storageClass: copyStringPointer(pvc.Spec.StorageClassName),
	}, nil
}

func equalStringPointers(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	copy := *value
	return &copy
}
