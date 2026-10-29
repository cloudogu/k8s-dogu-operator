package pvc

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DesiredSizes returns the rendered storage request of each live PVC the chart renders, standalone or from a claim template.
func DesiredSizes(renderedObjects []*unstructured.Unstructured, targetNamespace string, livePVCs []corev1.PersistentVolumeClaim) (map[client.ObjectKey]resource.Quantity, error) {
	desiredClaims, desiredTemplates, err := extractDesiredClaims(renderedObjects, targetNamespace)
	if err != nil {
		return nil, err
	}

	sizes := map[client.ObjectKey]resource.Quantity{}
	for i := range livePVCs {
		key := client.ObjectKeyFromObject(&livePVCs[i])
		for _, claim := range desiredClaims {
			if claim.key == key {
				sizes[key] = claim.storage.DeepCopy()
			}
		}
		for _, template := range desiredTemplates {
			if template.statefulSet.Namespace == key.Namespace && MatchesStatefulSetClaim(key.Name, template.name, template.statefulSet.Name) {
				sizes[key] = template.storage.DeepCopy()
			}
		}
	}

	return sizes, nil
}
