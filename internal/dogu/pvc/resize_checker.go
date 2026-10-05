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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResizeRequest describes a required PVC expansion.
type ResizeRequest struct {
	PVC     client.ObjectKey
	Desired resource.Quantity
}

// CheckResult contains only PVCs that require an expansion.
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
	var checkErr error
	for _, desired := range desiredClaims {
		request, include, err := c.checkPVC(ctx, desired)
		if err != nil {
			checkErr = errors.Join(checkErr, err)
			continue
		}
		if include {
			result.ResizeRequests = append(result.ResizeRequests, request)
		}
	}
	if checkErr != nil {
		return CheckResult{}, checkErr
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

	request := ResizeRequest{
		PVC:     desired.key,
		Desired: desired.storage.DeepCopy(),
	}

	var validationErr error
	if desired.storageClass != nil && !ptr.Equal(desired.storageClass, livePVC.Spec.StorageClassName) {
		validationErr = &StorageClassImmutableError{
			PVC:     desired.key,
			Current: copyStringPointer(livePVC.Spec.StorageClassName),
			Desired: copyStringPointer(desired.storageClass),
		}
	}

	currentRequest, err := currentStorageRequest(livePVC)
	if err != nil {
		return ResizeRequest{}, false, errors.Join(validationErr, err)
	}

	if desired.storage.Cmp(currentRequest) < 0 {
		validationErr = errors.Join(validationErr, &VolumeShrinkError{
			PVC:     desired.key,
			Current: currentRequest.DeepCopy(),
			Desired: desired.storage.DeepCopy(),
		})
	}
	if validationErr != nil {
		return ResizeRequest{}, false, validationErr
	}
	if desired.storage.Cmp(currentRequest) > 0 {
		return request, true, nil
	}

	return ResizeRequest{}, false, nil
}

func currentStorageRequest(pvc *corev1.PersistentVolumeClaim) (resource.Quantity, error) {
	key := client.ObjectKeyFromObject(pvc)
	if pvc.Status.Phase != corev1.ClaimBound {
		return resource.Quantity{}, fmt.Errorf("live PVC %q is not bound: phase %q", key, pvc.Status.Phase)
	}

	if resizeStatus := pvc.Status.AllocatedResourceStatuses[corev1.ResourceStorage]; resizeStatus != "" {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has storage resize status %q", key, resizeStatus)
	}

	for _, condition := range pvc.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && isResizeCondition(condition.Type) {
			return resource.Quantity{}, fmt.Errorf("live PVC %q has active resize condition %q", key, condition.Type)
		}
	}

	requested, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has no storage request", key)
	}
	capacity, found := pvc.Status.Capacity[corev1.ResourceStorage]
	if !found {
		return resource.Quantity{}, fmt.Errorf("live PVC %q has no storage capacity", key)
	}
	if requested.Cmp(capacity) > 0 {
		return resource.Quantity{}, fmt.Errorf("live PVC %q is not ready: storage request %s exceeds capacity %s", key, requested.String(), capacity.String())
	}

	return requested, nil
}

func isResizeCondition(conditionType corev1.PersistentVolumeClaimConditionType) bool {
	switch conditionType {
	case corev1.PersistentVolumeClaimResizing,
		corev1.PersistentVolumeClaimFileSystemResizePending,
		corev1.PersistentVolumeClaimControllerResizeError,
		corev1.PersistentVolumeClaimNodeResizeError:
		return true
	default:
		return false
	}
}

func extractDesiredClaims(renderedObjects []*unstructured.Unstructured) ([]desiredClaim, error) {
	var claims []desiredClaim
	var extractionErr error

	for _, unstructuredObject := range renderedObjects {
		switch unstructuredObject.GroupVersionKind().GroupKind() {
		case schema.GroupKind{Group: corev1.GroupName, Kind: "PersistentVolumeClaim"}:
			pvc := &corev1.PersistentVolumeClaim{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, pvc); err != nil {
				extractionErr = errors.Join(extractionErr, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err))
				continue
			}
			if pvc.Namespace == "" {
				pvc.Namespace = corev1.NamespaceDefault
			}

			claim, err := desiredClaimFromPVC(pvc)
			if err != nil {
				extractionErr = errors.Join(extractionErr, err)
				continue
			}
			claims = append(claims, claim)
		case schema.GroupKind{Group: appsv1.GroupName, Kind: "StatefulSet"}:
			statefulSet := &appsv1.StatefulSet{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, statefulSet); err != nil {
				extractionErr = errors.Join(extractionErr, fmt.Errorf("failed to convert rendered %s %q: %w", unstructuredObject.GetKind(), client.ObjectKeyFromObject(unstructuredObject), err))
				continue
			}
			if statefulSet.Namespace == "" {
				statefulSet.Namespace = corev1.NamespaceDefault
			}

			statefulSetClaims, err := desiredClaimsFromStatefulSet(statefulSet)
			if err != nil {
				extractionErr = errors.Join(extractionErr, err)
				continue
			}
			claims = append(claims, statefulSetClaims...)
		}
	}

	return claims, extractionErr
}

func desiredClaimsFromStatefulSet(statefulSet *appsv1.StatefulSet) ([]desiredClaim, error) {
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
	var extractionErr error
	for _, template := range statefulSet.Spec.VolumeClaimTemplates {
		for ordinal := startOrdinal; ordinal < startOrdinal+replicas; ordinal++ {
			pvc := template.DeepCopy()
			pvc.Name = fmt.Sprintf("%s-%s-%d", template.Name, statefulSet.Name, ordinal)
			pvc.Namespace = statefulSet.Namespace

			claim, err := desiredClaimFromPVC(pvc)
			if err != nil {
				extractionErr = errors.Join(extractionErr, err)
				continue
			}
			claims = append(claims, claim)
		}
	}

	return claims, extractionErr
}

func desiredClaimFromPVC(pvc *corev1.PersistentVolumeClaim) (desiredClaim, error) {
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == "" {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q uses unsupported static provisioning", client.ObjectKeyFromObject(pvc))
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

func copyStringPointer(value *string) *string {
	if value == nil {
		return nil
	}

	return ptr.To(*value)
}
