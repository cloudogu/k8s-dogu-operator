package pvc

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const growerFieldOwner = "k8s-dogu-operator-pvc-grower"

// GrownClaim is a claim whose storage request was raised from Previous to Desired.
type GrownClaim struct {
	PVC      client.ObjectKey
	Previous resource.Quantity
	Desired  resource.Quantity
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
		if err != nil {
			err = fmt.Errorf("failed to grow PVC %q to %s: %w", request.PVC, request.Desired.String(), err)
			if apierrors.IsForbidden(err) || apierrors.IsInvalid(err) {
				return grown, &GrowRejectedError{PVC: request.PVC, Err: err}
			}
			return grown, err
		}
		grown = append(grown, claim)
	}

	return grown, nil
}

func (g *Grower) grow(ctx context.Context, request ResizeRequest) (GrownClaim, error) {
	claim := corev1ac.PersistentVolumeClaim(request.PVC.Name, request.PVC.Namespace).
		WithSpec(corev1ac.PersistentVolumeClaimSpec().
			WithResources(corev1ac.VolumeResourceRequirements().
				WithRequests(corev1.ResourceList{corev1.ResourceStorage: request.Desired.DeepCopy()})))
	if err := g.k8s.Apply(ctx, claim, client.FieldOwner(growerFieldOwner), client.ForceOwnership); err != nil {
		return GrownClaim{}, fmt.Errorf("failed to apply PVC: %w", err)
	}

	return GrownClaim{PVC: request.PVC, Previous: request.Current.DeepCopy(), Desired: request.Desired.DeepCopy()}, nil
}
