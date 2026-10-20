package pending

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	helmflux "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sdk "helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// PendingChange describes whether desired input still needs to be applied.
type PendingChange struct {
	Status  metav1.ConditionStatus
	Message string
}

// ToCondition turns the combined result into the dogu's UpdatePending condition.
func (change PendingChange) ToCondition(reason string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type: v3beta1.ConditionUpdatePending, Status: change.Status, Reason: reason,
		Message: change.Message, ObservedGeneration: generation,
	}
}

// CombinePendingChanges returns True if any change is pending, Unknown if a comparison is incomplete, otherwise False.
func CombinePendingChanges(changes ...PendingChange) PendingChange {
	result := PendingChange{Status: metav1.ConditionFalse, Message: "No changes are pending"}
	for _, change := range changes {
		if change.Status == metav1.ConditionTrue {
			return change
		}
		if change.Status == metav1.ConditionUnknown {
			result = change
		}
	}
	return result
}

// PendingReleaseChange checks whether the desired HelmRelease still needs to be applied.
// If desired is nil, the desired HelmRelease could not be built from the chart and values.
func PendingReleaseChange(actual, desired *helmflux.HelmRelease, stored *sdk.Release, chartVersion string) PendingChange {
	changed := PendingChange{Status: metav1.ConditionTrue, Message: "HelmRelease changes are waiting to be applied"}
	unknown := PendingChange{Status: metav1.ConditionUnknown, Message: "The desired HelmRelease cannot yet be fully compared"}
	// Without a Helm release history, the initial installation is still pending.
	if actual == nil || len(actual.Status.History) == 0 {
		return PendingChange{Status: metav1.ConditionTrue, Message: "The dogu has not been installed yet"}
	}
	// If the latest Helm revision uses a different chart version than the dogu requests, an update is pending.
	latest := actual.Status.History.Latest()
	version, _, _ := strings.Cut(latest.ChartVersion, "+")
	if version != chartVersion {
		return changed
	}
	// Without the desired HelmRelease, we cannot compare the remaining settings.
	if desired == nil {
		return unknown
	}
	// Check whether Flux points to the desired chart source.
	if !reflect.DeepEqual(desired.Spec.ChartRef, actual.Spec.ChartRef) {
		return changed
	}
	// Check whether Flux has the desired install and upgrade settings.
	if !reflect.DeepEqual(desired.Spec.Install, actual.Spec.Install) || !reflect.DeepEqual(desired.Spec.Upgrade, actual.Spec.Upgrade) {
		return changed
	}
	// Check whether the HelmRelease contains the desired shared labels and annotations.
	if !reflect.DeepEqual(desired.Spec.CommonMetadata, actual.Spec.CommonMetadata) {
		return changed
	}
	// Check whether the HelmRelease contains the desired values.
	valuesEqual, err := equalValues(desired.GetValues(), actual.GetValues())
	if err != nil {
		return unknown
	}
	if !valuesEqual {
		return changed
	}
	// Without the stored Helm release, we cannot check what Helm has applied.
	if stored == nil {
		return unknown
	}
	// The HelmRelease may already contain new values while Helm still has the old ones.
	valuesEqual, err = equalValues(desired.GetValues(), stored.Config)
	if err != nil {
		return unknown
	}
	if !valuesEqual {
		return changed
	}
	// If the latest install or upgrade did not succeed, some resources may already have changed.
	if latest.Status != string(sdk.StatusDeployed) || stored.Info == nil || stored.Info.Status != sdk.StatusDeployed {
		return unknown
	}
	// Flux must report a completed release with no reconciliation in progress.
	if !meta.IsStatusConditionTrue(actual.Status.Conditions, helmflux.ReleasedCondition) || meta.IsStatusConditionTrue(actual.Status.Conditions, fluxmeta.ReconcilingCondition) {
		return unknown
	}
	// Check whether the stored manifest also contains the desired shared labels and annotations.
	metadataApplied, err := commonMetadataApplied(stored.Manifest, desired.Spec.CommonMetadata)
	if err != nil {
		return unknown
	}
	if !metadataApplied {
		return changed
	}
	// All checks passed: the desired release is deployed and no changes are pending.
	return PendingChange{Status: metav1.ConditionFalse, Message: "The desired HelmRelease is deployed"}
}

// equalValues compares JSON values independent of map key order and treats nil and empty maps as equal.
func equalValues(left, right map[string]any) (bool, error) {
	if len(left) == 0 && len(right) == 0 {
		return true, nil
	}
	leftJSON, err := json.Marshal(left)
	if err != nil {
		return false, err
	}
	rightJSON, err := json.Marshal(right)
	if err != nil {
		return false, err
	}
	return bytes.Equal(leftJSON, rightJSON), nil
}

// commonMetadataApplied checks rendered resources, since Flux also records metadata digests after failed attempts.
func commonMetadataApplied(manifest string, metadata *helmflux.CommonMetadata) (bool, error) {
	if metadata == nil {
		return true, nil
	}
	// Read each resource from Helm's stored manifest, not from the live cluster.
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 4096)
	for {
		var resource unstructured.Unstructured
		if err := decoder.Decode(&resource); err != nil {
			if err == io.EOF {
				return true, nil
			}
			return false, err
		}
		if len(resource.Object) == 0 {
			continue
		}
		// Check the requested labels; additional labels are allowed.
		for key, value := range metadata.Labels {
			if resource.GetLabels()[key] != value {
				return false, nil
			}
		}
		// Check the requested annotations; additional annotations are allowed.
		for key, value := range metadata.Annotations {
			if resource.GetAnnotations()[key] != value {
				return false, nil
			}
		}
	}
}
