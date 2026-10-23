package flux

import (
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	fluxsource "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func reconcilingCondition() metav1.Condition {
	return metav1.Condition{Type: meta.ReconcilingCondition, Status: metav1.ConditionTrue, Reason: meta.ProgressingReason, Message: "Fulfilling prerequisites"}
}

func deployedRelease() *helmv2.HelmRelease {
	return &helmv2.HelmRelease{
		Name: "nexus", Generation: 1, ResourceVersion: "1",
		Status: helmv2.HelmReleaseStatus{
			ObservedGeneration: 1,
			Conditions:         []metav1.Condition{*readyCondition(metav1.ConditionTrue, helmv2.InstallSucceededReason, "")},
		},
	}
}

func TestIgnoreReconcilingPredicate_Update(t *testing.T) {
	tests := []struct {
		name   string
		modify func(old, updated *helmv2.HelmRelease)
		want   bool
	}{
		{
			name: "Reconciling condition added",
			modify: func(_, updated *helmv2.HelmRelease) {
				updated.Status.Conditions = append(updated.Status.Conditions, reconcilingCondition())
			},
			want: false,
		},
		{
			name: "Reconciling condition removed",
			modify: func(old, _ *helmv2.HelmRelease) {
				old.Status.Conditions = append(old.Status.Conditions, reconcilingCondition())
			},
			want: false,
		},
		{
			name: "Reconciling condition is the only condition",
			modify: func(old, updated *helmv2.HelmRelease) {
				old.Status.Conditions = nil
				updated.Status.Conditions = []metav1.Condition{reconcilingCondition()}
			},
			want: false,
		},
		{
			name: "Ready condition changed together with Reconciling",
			modify: func(_, updated *helmv2.HelmRelease) {
				updated.Status.Conditions = []metav1.Condition{*readyCondition(metav1.ConditionFalse, helmv2.UpgradeFailedReason, ""), reconcilingCondition()}
			},
			want: true,
		},
		{
			name: "generation changed",
			modify: func(_, updated *helmv2.HelmRelease) {
				updated.Generation++
			},
			want: true,
		},
		{
			name: "observed generation changed",
			modify: func(_, updated *helmv2.HelmRelease) {
				updated.Status.ObservedGeneration++
			},
			want: true,
		},
		{
			name: "deletion started",
			modify: func(_, updated *helmv2.HelmRelease) {
				now := metav1.Now()
				updated.DeletionTimestamp = &now
			},
			want: true,
		},
		{
			name: "other status field changed",
			modify: func(_, updated *helmv2.HelmRelease) {
				updated.Status.LastAttemptedRevision = "1.2.3"
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := deployedRelease()
			updated := old.DeepCopy()
			updated.ResourceVersion = "2"
			tt.modify(old, updated)

			got := IgnoreReconcilingPredicate().Update(event.UpdateEvent{ObjectOld: old, ObjectNew: updated})

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("should not modify the objects of the event", func(t *testing.T) {
		old := deployedRelease()
		updated := old.DeepCopy()
		updated.Status.Conditions = append(updated.Status.Conditions, reconcilingCondition())

		IgnoreReconcilingPredicate().Update(event.UpdateEvent{ObjectOld: old, ObjectNew: updated})

		assert.Len(t, updated.Status.Conditions, 2)
		assert.Equal(t, "1", updated.ResourceVersion)
	})

	t.Run("should pass objects which are no HelmRelease", func(t *testing.T) {
		for _, e := range []event.UpdateEvent{
			{ObjectOld: &fluxsource.OCIRepository{}, ObjectNew: &fluxsource.OCIRepository{}},
			{ObjectOld: nil, ObjectNew: deployedRelease()},
			{ObjectOld: deployedRelease(), ObjectNew: client.Object(nil)},
		} {
			assert.True(t, IgnoreReconcilingPredicate().Update(e))
		}
	})
}
