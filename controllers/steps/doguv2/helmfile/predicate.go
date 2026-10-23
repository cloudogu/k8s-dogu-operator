package helmfile

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LifecyclePredicate preserves the generation filter for ordinary Dogus and also
// admits deletion requests for resources protected by the Helmfile finalizer.
// This is needed because Helmfile dogus might be in namespaces that aren't watched by the operator
type LifecyclePredicate struct {
	predicate.GenerationChangedPredicate
}

func (p LifecyclePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	return p.GenerationChangedPredicate.Update(e) ||
		(controllerutil.ContainsFinalizer(e.ObjectNew, Finalizer) &&
			e.ObjectOld.GetDeletionTimestamp().IsZero() && !e.ObjectNew.GetDeletionTimestamp().IsZero())
}
