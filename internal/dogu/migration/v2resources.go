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

// deletableResourceKinds contains all the resource kinds the dogu-operator (possibly) manages for v2 dogus, so this
// is also the list of resource types to delete before a migration to v3.
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
}

// V2ResourceFinder finds all managed resources that are owned by a dogu.
type V2ResourceFinder struct {
	client client.Reader
}

// NewV2ResourceFinder creates a V2ResourceFinder.
func NewV2ResourceFinder(client client.Reader) *V2ResourceFinder {
	return &V2ResourceFinder{
		client: client,
	}
}

// FindV2Resources returns all resources managed by the dogu operator for a v2 dogu. It returns resources of all managed
// types, which have the label `dogu.name` set to the name of the given dogu reference, and which have an owner
// reference to that dogu set. Resources with the label `k8s.cloudogu.com/type` set to `migration` are explicitly excluded.
func (vrf V2ResourceFinder) FindV2Resources(ctx context.Context, dogu v1.ObjectReference) ([]client.Object, error) {
	var owned []client.Object

	for _, kind := range deletableResourceKinds {
		byKind, err := vrf.findOwnedByKind(ctx, kind, dogu)
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

	removeMigrationResources(list)

	var owned []client.Object
	for i := range list.Items {
		item := &list.Items[i]
		if !hasOwnerReference(item, ref) {
			continue
		}
		// depending on the client used to fetch the items, the GVK may or may not be set,
		// so we ensure it is set correctly
		item.SetGroupVersionKind(gvk)
		owned = append(owned, item)
	}

	return owned, nil
}

func removeMigrationResources(list *metav1.PartialObjectMetadataList) {
	list.Items = slices.DeleteFunc(list.Items, func(item metav1.PartialObjectMetadata) bool {
		return item.Labels[configTypeLabel] == migrationType
	})
}

func hasOwnerReference(obj metav1.Object, ref v1.ObjectReference) bool {
	return slices.ContainsFunc(obj.GetOwnerReferences(), func(owner metav1.OwnerReference) bool {
		return owner.UID == ref.UID
	})
}
