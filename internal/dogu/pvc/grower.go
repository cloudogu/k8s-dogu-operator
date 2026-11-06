package pvc

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GrownClaim is a claim whose storage request was raised from Previous.
type GrownClaim struct {
	PVC      corev1.PersistentVolumeClaim
	Previous resource.Quantity
}

// Grower raises the storage request of claims to their desired size.
type Grower struct {
	k8s client.Client
}

func NewGrower(k8s client.Client) *Grower {
	return &Grower{k8s: k8s}
}

// Grow raises the claims in order and stops at the first failure, returning the claims grown until then.
// A claim the cluster refuses to grow is returned as GrowRejectedError.
func (g *Grower) Grow(ctx context.Context, requests []ResizeRequest) ([]GrownClaim, error) {
	var grown []GrownClaim
	for _, request := range requests {
		claim, err := g.grow(ctx, request)
		if apierrors.IsForbidden(err) || apierrors.IsInvalid(err) {
			return grown, &GrowRejectedError{PVC: request.PVC, Err: err}
		}
		if err != nil {
			return grown, err
		}
		grown = append(grown, claim)
	}

	return grown, nil
}

func (g *Grower) grow(ctx context.Context, request ResizeRequest) (GrownClaim, error) {
	base := &corev1.PersistentVolumeClaim{Namespace: request.PVC.Namespace, Name: request.PVC.Name}
	claim := base.DeepCopy()
	claim.Spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: request.Desired.DeepCopy()}
	if err := g.k8s.Patch(ctx, claim, client.MergeFrom(base)); err != nil {
		return GrownClaim{}, fmt.Errorf("failed to grow PVC %q to %s: %w", request.PVC, request.Desired.String(), err)
	}

	return GrownClaim{PVC: *claim, Previous: request.Current.DeepCopy()}, nil
}
