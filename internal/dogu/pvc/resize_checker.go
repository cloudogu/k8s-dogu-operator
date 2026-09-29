package pvc

import (
	"context"

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
