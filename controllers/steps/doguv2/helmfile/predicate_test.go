package helmfile

import (
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func TestLifecyclePredicate(t *testing.T) {
	for _, kind := range []string{"ordinary", "helmfile", "annotation only", "unrelated finalizer"} {
		t.Run(kind, func(t *testing.T) {
			for _, scenario := range []string{"generation", "deletion started", "status", "finalizers", "already deleting", "missing old", "missing new"} {
				t.Run(scenario, func(t *testing.T) {
					old := &doguv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "dogu", Generation: 1}}
					switch kind {
					case "helmfile":
						old.Finalizers = []string{Finalizer}
					case "annotation only":
						old.Annotations = map[string]string{ManagedAnnotation: "true"}
					case "unrelated finalizer":
						old.Finalizers = []string{"k8s.cloudogu.com/dogu-cleanup"}
					}
					updated := old.DeepCopy()
					now := metav1.Now()
					want := false
					switch scenario {
					case "generation":
						updated.Generation++
						want = true
					case "deletion started":
						updated.DeletionTimestamp = &now
						want = kind == "helmfile"
					case "status":
						updated.Status.Health = doguv2.AvailableHealthStatus
					case "finalizers":
						updated.Finalizers = append(updated.Finalizers, "example.com/cleanup")
					case "already deleting":
						old.DeletionTimestamp = &now
						updated.DeletionTimestamp = &now
					}
					e := event.UpdateEvent{ObjectOld: old, ObjectNew: updated}
					if scenario == "missing old" {
						e.ObjectOld = nil
					}
					if scenario == "missing new" {
						e.ObjectNew = nil
					}
					assert.Equal(t, want, (LifecyclePredicate{}).Update(e))
					if kind != "helmfile" {
						assert.Equal(t, (predicate.GenerationChangedPredicate{}).Update(e), (LifecyclePredicate{}).Update(e))
					}
				})
			}
		})
	}
}
