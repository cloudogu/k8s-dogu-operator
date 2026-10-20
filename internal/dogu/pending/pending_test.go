package pending

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	helmflux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	sdk "helm.sh/helm/v3/pkg/release"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPendingChange_ToCondition(t *testing.T) {
	change := PendingChange{Status: metav1.ConditionTrue, Message: "A version update is pending"}

	assert.Equal(t, metav1.Condition{
		Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonStopped,
		Message: "A version update is pending", ObservedGeneration: 7,
	}, change.ToCondition(v3beta1.ReasonStopped, 7))
}

func TestPendingReleaseChange(t *testing.T) {
	values := `{"replicas":1,"feature":true}`
	deployed := &helmflux.HelmRelease{
		Spec: helmflux.HelmReleaseSpec{Suspend: true, Values: &apiext.JSON{Raw: []byte(values)}},
		Status: helmflux.HelmReleaseStatus{
			History:    helmflux.Snapshots{{Version: 1, Status: "deployed", ChartVersion: "1.2.3+build"}},
			Conditions: []metav1.Condition{{Type: helmflux.ReleasedCondition, Status: metav1.ConditionTrue}},
		},
	}
	// Each case starts with a deployed release and changes its input or stored state to check the pending result.
	for _, tc := range []struct {
		name   string
		change func(*helmflux.HelmRelease, *helmflux.HelmRelease, *sdk.Release)
		want   metav1.ConditionStatus
	}{
		{name: "suspension alone is not pending", want: metav1.ConditionFalse},
		{
			name: "empty and missing values are equivalent",
			change: func(actual, desired *helmflux.HelmRelease, stored *sdk.Release) {
				actual.Spec.Values = nil
				desired.Spec.Values = nil
				stored.Config = map[string]any{}
			},
			want: metav1.ConditionFalse,
		},
		{
			name: "values differ from deployed history even after upsert",
			change: func(actual, desired *helmflux.HelmRelease, _ *sdk.Release) {
				// Flux has the new input, but Helm's stored values remain unchanged.
				desired.Spec.Values.Raw = []byte(`{"replicas":2,"feature":true}`)
				actual.Spec.Values = desired.Spec.Values.DeepCopy()
			},
			want: metav1.ConditionTrue,
		},
		{
			name: "paused HelmRelease input differs from the dogu",
			change: func(actual, _ *helmflux.HelmRelease, _ *sdk.Release) {
				actual.Spec.Values.Raw = []byte(`{"replicas":3,"feature":true}`)
			},
			want: metav1.ConditionTrue,
		},
		{
			name: "failed attempt is not successful application",
			change: func(actual, _ *helmflux.HelmRelease, stored *sdk.Release) {
				// An older successful revision must not hide the newer failed upgrade.
				actual.Status.History = append(actual.Status.History, &helmflux.Snapshot{
					Version: 2, Status: "failed", ChartVersion: "1.2.3",
				})
				stored.Info.Status = sdk.StatusFailed
			},
			want: metav1.ConditionUnknown,
		},
		{
			name: "metadata changed",
			change: func(_, desired *helmflux.HelmRelease, _ *sdk.Release) {
				desired.Spec.CommonMetadata = &helmflux.CommonMetadata{Labels: map[string]string{"version": "1.2.3"}}
			},
			want: metav1.ConditionTrue,
		},
		{
			name: "stored manifest still has old labels",
			change: func(actual, desired *helmflux.HelmRelease, stored *sdk.Release) {
				desired.Spec.CommonMetadata = &helmflux.CommonMetadata{Labels: map[string]string{"version": "new"}}
				actual.Spec.CommonMetadata = desired.Spec.CommonMetadata
				stored.Manifest = `
apiVersion: v1
kind: Service
metadata:
  name: web
  labels:
    version: old
`
			},
			want: metav1.ConditionTrue,
		},
		{
			name: "stored metadata matches",
			change: func(actual, desired *helmflux.HelmRelease, stored *sdk.Release) {
				desired.Spec.CommonMetadata = &helmflux.CommonMetadata{
					Labels: map[string]string{"version": "new"}, Annotations: map[string]string{"owner": "dogu"},
				}
				actual.Spec.CommonMetadata = desired.Spec.CommonMetadata
				stored.Manifest = `
apiVersion: v1
kind: Service
metadata:
  name: web
  labels:
    version: new
  annotations:
    owner: dogu
`
			},
			want: metav1.ConditionFalse,
		},
		{
			name: "another resource is missing the annotation",
			change: func(actual, desired *helmflux.HelmRelease, stored *sdk.Release) {
				desired.Spec.CommonMetadata = &helmflux.CommonMetadata{Annotations: map[string]string{"owner": "dogu"}}
				actual.Spec.CommonMetadata = desired.Spec.CommonMetadata
				// The first resource matches; the comparison must also check the second one.
				stored.Manifest = `
apiVersion: v1
kind: Service
metadata:
  annotations:
    owner: dogu
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
`
			},
			want: metav1.ConditionTrue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual, desired := deployed.DeepCopy(), deployed.DeepCopy()
			stored := &sdk.Release{Config: deployed.GetValues(), Info: &sdk.Info{Status: sdk.StatusDeployed}}
			if tc.change != nil {
				tc.change(actual, desired, stored)
			}
			assert.Equal(t, tc.want, PendingReleaseChange(actual, desired, stored, "1.2.3").Status)
		})
	}

	// Missing data prevents a full comparison, but a different version or a missing installation still means changes are pending.
	for _, tc := range []struct {
		name            string
		actual, desired *helmflux.HelmRelease
		version         string
		want            metav1.ConditionStatus
	}{
		{name: "not installed", version: "1.2.3", want: metav1.ConditionTrue},
		{name: "desired input unavailable", actual: deployed, version: "1.2.3", want: metav1.ConditionUnknown},
		{name: "version change known without desired input", actual: deployed, version: "2.0.0", want: metav1.ConditionTrue},
		{name: "stored release unavailable", actual: deployed, desired: deployed, version: "1.2.3", want: metav1.ConditionUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, PendingReleaseChange(tc.actual, tc.desired, nil, tc.version).Status)
		})
	}
}

func TestCombinePendingChanges(t *testing.T) {
	// Return True if either check finds a change, Unknown if a check cannot finish, and False if both find no changes.
	for _, tc := range []struct {
		name               string
		oci, release, want metav1.ConditionStatus
	}{
		{"source change is pending", metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionTrue},
		{"release change wins over unknown source", metav1.ConditionUnknown, metav1.ConditionTrue, metav1.ConditionTrue},
		{"release comparison is incomplete", metav1.ConditionFalse, metav1.ConditionUnknown, metav1.ConditionUnknown},
		{"source comparison is incomplete", metav1.ConditionUnknown, metav1.ConditionFalse, metav1.ConditionUnknown},
		{"both comparisons match", metav1.ConditionFalse, metav1.ConditionFalse, metav1.ConditionFalse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, CombinePendingChanges(PendingChange{Status: tc.oci}, PendingChange{Status: tc.release}).Status)
		})
	}
}
