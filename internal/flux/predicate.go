package flux

import (
	"slices"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ignoredConditionTypes lists the conditions to ignore.
func ignoredConditionTypes() []string {
	return []string{meta.ReconcilingCondition}
}

// IgnoreReconcilingPredicate drops updates of a HelmRelease which only differ in ignored conditions.
func IgnoreReconcilingPredicate() predicate.Predicate {
	return predicate.Funcs{UpdateFunc: func(e event.UpdateEvent) bool {
		oldRelease, okOld := e.ObjectOld.(*helmv2.HelmRelease)
		newRelease, okNew := e.ObjectNew.(*helmv2.HelmRelease)
		if !okOld || !okNew {
			return true
		}

		return !equality.Semantic.DeepEqual(withoutIgnoredFields(oldRelease), withoutIgnoredFields(newRelease))
	}}
}

// withoutIgnoredFields returns a copy of the release without the fields the predicate ignores.
func withoutIgnoredFields(release *helmv2.HelmRelease) *helmv2.HelmRelease {
	// The HelmRelease is copied, because the event objects point into the informer cache and must not be modified.
	stripped := release.DeepCopy()
	stripped.ResourceVersion = ""
	stripped.ManagedFields = nil
	ignored := ignoredConditionTypes()
	stripped.Status.Conditions = slices.DeleteFunc(stripped.Status.Conditions, func(c metav1.Condition) bool {
		return slices.Contains(ignored, c.Type)
	})
	return stripped
}
