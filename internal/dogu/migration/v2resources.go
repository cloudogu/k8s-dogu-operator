package migration

import (
	"context"
	"fmt"
	"slices"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var deletableResourceKinds = []schema.GroupVersionKind{
	{Version: "v1", Kind: "ConfigMap"},
	{Version: "v1", Kind: "Secret"},
	{Version: "v1", Kind: "Service"},
	{Version: "v1", Kind: "PersistentVolumeClaim"},
	{Group: "apps", Version: "v1", Kind: "Deployment"},
	{Group: "k8s.cloudogu.com", Version: "v1", Kind: "AuthRegistration"},
	{Group: "k8s.cloudogu.com", Version: "v1", Kind: "Exposition"},
	{Group: "k8s.cloudogu.com", Version: "v1", Kind: "WarpMenuEntry"},
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"},
	// TODO: AuthRegistration & Exposition seems to lack the proper dogu.name label (or any others)
	// TODO: what about any existing DoguRestart resources?
}

// V2ResourceFinder finds all resources that are owned by a dogu.
type V2ResourceFinder struct {
	client client.Reader
}

// NewV2ResourceFinder creates a V2ResourceFinder. The client should read directly from the API server (not from a
// cache), because resources of arbitrary kinds are listed.
func NewV2ResourceFinder(client client.Reader) V2ResourceFinder {
	return V2ResourceFinder{
		client: client,
	}
}

// FindV2Resources returns all resources in the cluster that have an owner reference to the given object.
// Kubernetes offers no way to select resources by their owner. Therefore, all listable namespaced resource types of
// the cluster are discovered and listed (metadata only) in the namespace of the owner. The resulting objects
// are of type metav1.PartialObjectMetadata and contain the type information needed to delete them.
func (vrf V2ResourceFinder) FindV2Resources(ctx context.Context, ref v1.ObjectReference) ([]client.Object, error) {
	var owned []client.Object

	for _, kind := range deletableResourceKinds {
		byKind, err := vrf.findOwnedByKind(ctx, kind, ref)
		if err != nil {
			return nil, err
		}
		owned = append(owned, byKind...)
	}

	return owned, nil
}

func (vrf V2ResourceFinder) findOwnedByKind(ctx context.Context, gvk schema.GroupVersionKind, ref v1.ObjectReference) ([]client.Object, error) {
	list := &metav1.PartialObjectMetadataList{}
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))

	if err := vrf.client.List(ctx, list, client.InNamespace(ref.Namespace), client.MatchingLabels{
		"dogu.name": ref.Name,
	}); err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", gvk, err)
	}

	var owned []client.Object
	for i := range list.Items {
		item := &list.Items[i]
		if !hasOwnerReference(item, ref) {
			continue
		}
		item.SetGroupVersionKind(gvk)
		owned = append(owned, item)
	}

	return owned, nil
}

func hasOwnerReference(obj metav1.Object, ref v1.ObjectReference) bool {
	return slices.ContainsFunc(obj.GetOwnerReferences(), func(owner metav1.OwnerReference) bool {
		return owner.UID == ref.UID
	})
}
