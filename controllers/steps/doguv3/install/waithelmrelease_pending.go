package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	helmflux "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	sdk "helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type pendingChange struct {
	Status  metav1.ConditionStatus
	Message string
}

func (hrs *HelmReleaseStatusStep) checkPendingChanges(ctx context.Context, dogu *v3beta1.Dogu, release *helmflux.HelmRelease) stepsv3.StepResult {
	// OCI detection runs first. A known difference already protects this generation
	// from the final reset; do not assemble against the previous chart during pause.
	condition := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	if condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == dogu.Generation {
		return stepsv3.Continue()
	}
	change, err := pendingReleaseChange(release, nil, nil, dogu.Spec.Version)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}
	if change.Status != metav1.ConditionTrue {
		values, err := combineValues(ctx, dogu, hrs.chartService, hrs.assembler)
		if err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
		}
		desired := &helmflux.HelmRelease{}
		configureHelmRelease(desired, dogu, values, &hrs.retryInterval, &hrs.helmReconcileInterval)
		change, err = pendingReleaseChange(release, desired, nil, dogu.Spec.Version)
		if err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
		}
		if change.Status == metav1.ConditionUnknown {
			stored, err := hrs.releaseReader.StoredRelease(release)
			if err != nil {
				return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
			}
			change, err = pendingReleaseChange(release, desired, stored, dogu.Spec.Version)
			if err != nil {
				return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
			}
		}
	}
	if change.Status == metav1.ConditionUnknown {
		// Reaching the reset step must mean the comparison completed, not merely
		// that this step found no definite difference.
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, change.Message)
	}
	if change.Status == metav1.ConditionTrue {
		reason := v3beta1.ReasonStopped
		if dogu.Spec.PauseReconciliation {
			reason = v3beta1.ReasonReconciliationPaused
		}
		if meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
			Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionTrue, Reason: reason,
			Message: change.Message, ObservedGeneration: dogu.Generation,
		}) {
			if err := hrs.k8sClient.Status().Update(ctx, dogu); err != nil {
				return stepsv3.RequeueWithError(fmt.Errorf("failed to report pending HelmRelease changes: %w", err), reason)
			}
		}
	}
	// Only the final reset step writes False, after all relevant checks complete.
	return stepsv3.Continue()
}

// pendingReleaseChange checks both desired Flux input and what Helm has applied.
func pendingReleaseChange(actual, desired *helmflux.HelmRelease, stored *sdk.Release, chartVersion string) (pendingChange, error) {
	changed := pendingChange{Status: metav1.ConditionTrue, Message: "HelmRelease changes are waiting to be applied"}
	unknown := pendingChange{Status: metav1.ConditionUnknown, Message: "The desired HelmRelease cannot yet be fully compared"}
	if actual == nil || len(actual.Status.History) == 0 {
		return pendingChange{Status: metav1.ConditionTrue, Message: "The dogu has not been installed yet"}, nil
	}
	latest := actual.Status.History.Latest()
	version, _, _ := strings.Cut(latest.ChartVersion, "+")
	if version != chartVersion {
		return changed, nil
	}
	if desired == nil {
		return unknown, nil
	}
	inputMatches, err := releaseInputMatches(actual, desired)
	if err != nil {
		return unknown, fmt.Errorf("failed to compare inputs for HelmRelease %s/%s: %w", actual.Namespace, actual.Name, err)
	}
	if !inputMatches {
		return changed, nil
	}
	if stored == nil {
		return unknown, nil
	}
	valuesEqual, err := equalValues(desired.GetValues(), stored.Config)
	if err != nil {
		return unknown, fmt.Errorf("failed to compare stored values for HelmRelease %s/%s: %w", actual.Namespace, actual.Name, err)
	}
	if !valuesEqual {
		return changed, nil
	}
	if latest.Status != string(sdk.StatusDeployed) || stored.Info == nil || stored.Info.Status != sdk.StatusDeployed {
		return unknown, nil
	}
	if !meta.IsStatusConditionTrue(actual.Status.Conditions, helmflux.ReleasedCondition) || meta.IsStatusConditionTrue(actual.Status.Conditions, fluxmeta.ReconcilingCondition) {
		return unknown, nil
	}
	metadataApplied, err := commonMetadataApplied(stored.Manifest, desired.Spec.CommonMetadata)
	if err != nil {
		return unknown, fmt.Errorf("failed to decode stored manifest for HelmRelease %s/%s: %w", actual.Namespace, actual.Name, err)
	}
	if !metadataApplied {
		return changed, nil
	}
	return pendingChange{Status: metav1.ConditionFalse, Message: "The desired HelmRelease is deployed"}, nil
}

func releaseInputMatches(actual, desired *helmflux.HelmRelease) (bool, error) {
	if !reflect.DeepEqual(desired.Spec.ChartRef, actual.Spec.ChartRef) {
		return false, nil
	}
	if !reflect.DeepEqual(desired.Spec.Install, actual.Spec.Install) || !reflect.DeepEqual(desired.Spec.Upgrade, actual.Spec.Upgrade) {
		return false, nil
	}
	if !reflect.DeepEqual(desired.Spec.CommonMetadata, actual.Spec.CommonMetadata) {
		return false, nil
	}
	return equalValues(desired.GetValues(), actual.GetValues())
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

// commonMetadataApplied checks Helm's stored manifest, not the live workloads.
func commonMetadataApplied(manifest string, metadata *helmflux.CommonMetadata) (bool, error) {
	if metadata == nil {
		return true, nil
	}
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
		if !containsMetadata(resource.GetLabels(), metadata.Labels) || !containsMetadata(resource.GetAnnotations(), metadata.Annotations) {
			return false, nil
		}
	}
}

func containsMetadata(actual, desired map[string]string) bool {
	for key, value := range desired {
		if actual[key] != value {
			return false
		}
	}
	return true
}
