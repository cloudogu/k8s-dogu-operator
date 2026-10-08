package flux

import (
	"testing"

	helmflux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testChartVersion      = "3.86.2-7"
	testChartVersionBuild = "3.86.2-7+ebb48ffcdce3"
	testOldChartVersion   = "3.86.2-6+0123456789ab"
)

func TestNewDesiredRelease(t *testing.T) {
	release := &helmflux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Generation: 3}}

	desired := NewDesiredRelease("3.86.2-8", release)

	assert.Equal(t, DesiredRelease{ChartVersion: "3.86.2-8", Generation: 3}, desired)
}

func snapshot(version int, status, chartVersion string) *helmflux.Snapshot {
	return &helmflux.Snapshot{Version: version, Status: status, ChartVersion: chartVersion}
}

func readyCondition(status metav1.ConditionStatus, reason, msg string) *metav1.Condition {
	return &metav1.Condition{Type: meta.ReadyCondition, Status: status, Reason: reason, Message: msg}
}

func release(generation, observedGeneration int64, ready *metav1.Condition, history ...*helmflux.Snapshot) *helmflux.HelmRelease {
	hr := &helmflux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Generation: generation}}
	hr.Status.ObservedGeneration = observedGeneration
	hr.Status.History = history
	if ready != nil {
		hr.Status.Conditions = []metav1.Condition{*ready}
	}
	return hr
}

func TestEvaluateRelease(t *testing.T) {
	tests := []struct {
		name        string
		release     *helmflux.HelmRelease
		wantPhase   ReleasePhase
		wantMessage string
	}{
		{
			name:        "chart unavailable if there is no history and the artifact failed",
			release:     release(1, 1, readyCondition(metav1.ConditionFalse, helmflux.ArtifactFailedReason, "artifact not found")),
			wantPhase:   PhaseChartUnavailable,
			wantMessage: "artifact not found",
		},
		{
			name:        "installing while the first install runs",
			release:     release(1, -1, readyCondition(metav1.ConditionUnknown, meta.ProgressingReason, "reconciliation in progress")),
			wantPhase:   PhaseInstalling,
			wantMessage: "waiting for the helm-controller to apply generation 1 of the HelmRelease",
		},
		{
			name:      "installing if the HelmRelease was not processed at all yet",
			release:   release(1, 0, nil),
			wantPhase: PhaseInstalling,
		},
		{
			name: "upgrading if the spec changed and the helm-controller has not started yet (stale Ready=True)",
			release: release(2, 1, readyCondition(metav1.ConditionTrue, helmflux.InstallSucceededReason, ""),
				snapshot(1, statusDeployed, testChartVersionBuild)),
			wantPhase: PhaseUpgrading,
		},
		{
			name: "upgrading if the new revision is written but observedGeneration still lags behind",
			release: release(2, 1, readyCondition(metav1.ConditionTrue, helmflux.UpgradeSucceededReason, ""),
				snapshot(2, statusDeployed, testChartVersionBuild), snapshot(1, statusSuperseded, testChartVersionBuild)),
			wantPhase: PhaseUpgrading,
		},
		{
			name: "installing while a previously failed install is retried after a spec change",
			release: release(2, 1, readyCondition(metav1.ConditionFalse, helmflux.UpgradeFailedReason, "old error"),
				snapshot(1, "failed", testChartVersionBuild)),
			wantPhase: PhaseInstalling,
		},
		{
			name: "install failed if the install failed and was retried as upgrade without any deployed revision",
			release: release(1, 1, readyCondition(metav1.ConditionFalse, helmflux.UpgradeFailedReason, "stalled resources"),
				snapshot(3, "failed", testChartVersionBuild), snapshot(2, "failed", testChartVersionBuild), snapshot(1, "failed", testChartVersionBuild)),
			wantPhase:   PhaseInstallFailed,
			wantMessage: "stalled resources",
		},
		{
			name: "upgrade failed if a revision was deployed before",
			release: release(2, 2, readyCondition(metav1.ConditionFalse, helmflux.UpgradeFailedReason, "upgrade error"),
				snapshot(2, "failed", testChartVersionBuild), snapshot(1, statusDeployed, testOldChartVersion)),
			wantPhase:   PhaseUpgradeFailed,
			wantMessage: "upgrade error",
		},
		{
			name: "upgrade failed without a new revision, e.g. a render error",
			release: release(2, 2, readyCondition(metav1.ConditionFalse, helmflux.UpgradeFailedReason, "render error"),
				snapshot(1, statusDeployed, testChartVersionBuild)),
			wantPhase:   PhaseUpgradeFailed,
			wantMessage: "render error",
		},
		{
			name: "deployed if the current spec is processed and the desired chart version is deployed",
			release: release(1, 1, readyCondition(metav1.ConditionTrue, helmflux.InstallSucceededReason, ""),
				snapshot(1, statusDeployed, testChartVersionBuild)),
			wantPhase:   PhaseDeployed,
			wantMessage: "release revision 1 with chart version 3.86.2-7 is deployed",
		},
		{
			name: "deployed with a chart version without build metadata",
			release: release(1, 1, readyCondition(metav1.ConditionTrue, helmflux.InstallSucceededReason, ""),
				snapshot(1, statusDeployed, testChartVersion)),
			wantPhase: PhaseDeployed,
		},
		{
			name: "upgrading if the deployed chart version does not match the desired one",
			release: release(2, 2, readyCondition(metav1.ConditionTrue, helmflux.UpgradeSucceededReason, ""),
				snapshot(2, statusDeployed, testOldChartVersion)),
			wantPhase:   PhaseUpgrading,
			wantMessage: "waiting for the helm-controller to deploy chart version 3.86.2-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := DesiredRelease{ChartVersion: testChartVersion, Generation: tt.release.Generation}

			got := EvaluateRelease(tt.release, desired)

			assert.Equal(t, tt.wantPhase, got.Phase)
			if tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, got.Message)
			}
		})
	}
}

func TestTrimBuildMetadata(t *testing.T) {
	assert.Equal(t, "3.86.2-7-dev.1791444074", trimBuildMetadata("3.86.2-7-dev.1791444074+ebb48ffcdce3"))
	assert.Equal(t, "3.86.2-7", trimBuildMetadata("3.86.2-7"))
}
