package pvc

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	Check(ctx context.Context, renderedObjects []client.Object) (CheckResult, error)
}

type desiredClaim struct {
	key     client.ObjectKey
	storage resource.Quantity
}

func extractDesiredClaims(renderedObjects []client.Object) ([]desiredClaim, error) {
	var claims []desiredClaim

	for _, object := range renderedObjects {
		pvc, ok := object.(*corev1.PersistentVolumeClaim)
		if !ok {
			continue
		}

		claim, err := desiredClaimFromPVC(pvc)
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}

	return claims, nil
}

func desiredClaimFromPVC(pvc *corev1.PersistentVolumeClaim) (desiredClaim, error) {
	storage, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		return desiredClaim{}, fmt.Errorf("rendered PVC %q has no storage request", client.ObjectKeyFromObject(pvc))
	}

	return desiredClaim{
		key:     client.ObjectKeyFromObject(pvc),
		storage: storage.DeepCopy(),
	}, nil
}
