package flux

import (
	"fmt"
	"strings"

	helmflux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/fluxcd/pkg/apis/meta"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	statusDeployed   = "deployed"
	statusSuperseded = "superseded"
)

// ReleasePhase describes the state of a dogu's HelmRelease in the operator's own vocabulary.
type ReleasePhase string

const (
	// PhaseChartUnavailable means no release exists yet because the helm-controller cannot get the chart artifact.
	PhaseChartUnavailable ReleasePhase = "ChartUnavailable"
	// PhaseInstalling means the helm-controller has not finished the current spec and no revision was ever deployed.
	PhaseInstalling ReleasePhase = "Installing"
	// PhaseUpgrading means the helm-controller has not finished the current spec and a revision was deployed before.
	PhaseUpgrading ReleasePhase = "Upgrading"
	// PhaseInstallFailed means the attempt for the current spec failed and no revision was ever deployed.
	PhaseInstallFailed ReleasePhase = "InstallFailed"
	// PhaseUpgradeFailed means the attempt for the current spec failed and a revision was deployed before.
	PhaseUpgradeFailed ReleasePhase = "UpgradeFailed"
	// PhaseDeployed means the current spec is deployed with the desired chart version.
	PhaseDeployed ReleasePhase = "Deployed"
)

// ReleaseState is the evaluated state of a dogu's HelmRelease.
type ReleaseState struct {
	Phase ReleasePhase
	// Message describes the state
	Message string
}

// DesiredRelease describes the release state the operator expects the helm-controller to apply for a dogu.
type DesiredRelease struct {
	// ChartVersion is the chart version requested by the dogu resource.
	ChartVersion string
	// Generation is the generation of the HelmRelease spec the operator wrote last.
	// It changes with every spec change, e.g. a new chart version or new values.
	Generation int64
}

// NewDesiredRelease builds the desired release from the dogu's requested chart version and the current HelmRelease.
func NewDesiredRelease(chartVersion string, release *helmflux.HelmRelease) DesiredRelease {
	return DesiredRelease{
		ChartVersion: chartVersion,
		Generation:   release.Generation,
	}
}

// EvaluateRelease derives the phase of the dogu's HelmRelease from its status and the desired release.
func EvaluateRelease(release *helmflux.HelmRelease, desired DesiredRelease) ReleaseState {
	ready := apimeta.FindStatusCondition(release.Status.Conditions, meta.ReadyCondition)
	everDeployed := hasEverBeenDeployed(release.Status.History)

	if len(release.Status.History) == 0 && hasReason(ready, metav1.ConditionFalse, helmflux.ArtifactFailedReason) {
		return ReleaseState{Phase: PhaseChartUnavailable, Message: ready.Message}
	}

	if release.Status.ObservedGeneration < desired.Generation {
		msg := fmt.Sprintf("waiting for the helm-controller to apply generation %d of the HelmRelease", desired.Generation)
		return inProgress(everDeployed, msg)
	}

	if hasReason(ready, metav1.ConditionFalse, helmflux.InstallFailedReason, helmflux.UpgradeFailedReason) {
		return failed(everDeployed, ready.Message)
	}

	latest := release.Status.History.Latest()
	if latest != nil && latest.Status == statusDeployed {
		deployedVersion := trimBuildMetadata(latest.ChartVersion)
		if deployedVersion == desired.ChartVersion {
			msg := fmt.Sprintf("release revision %d with chart version %s is deployed", latest.Version, deployedVersion)
			return ReleaseState{Phase: PhaseDeployed, Message: msg}
		}
	}

	msg := fmt.Sprintf("waiting for the helm-controller to deploy chart version %s", desired.ChartVersion)
	return inProgress(everDeployed, msg)
}

func inProgress(everDeployed bool, msg string) ReleaseState {
	if everDeployed {
		return ReleaseState{Phase: PhaseUpgrading, Message: msg}
	}
	return ReleaseState{Phase: PhaseInstalling, Message: msg}
}

func failed(everDeployed bool, msg string) ReleaseState {
	if everDeployed {
		return ReleaseState{Phase: PhaseUpgradeFailed, Message: msg}
	}
	return ReleaseState{Phase: PhaseInstallFailed, Message: msg}
}

func hasEverBeenDeployed(history helmflux.Snapshots) bool {
	for _, hist := range history {
		if hist.Status == statusDeployed || hist.Status == statusSuperseded {
			return true
		}
	}
	return false
}

func hasReason(condition *metav1.Condition, status metav1.ConditionStatus, reasons ...string) bool {
	if condition == nil || condition.Status != status {
		return false
	}
	for _, reason := range reasons {
		if condition.Reason == reason {
			return true
		}
	}
	return false
}

// trimBuildMetadata removes the SemVer build metadata the helm-controller appends to the chart version of OCI charts,
// e.g. "3.86.2-7+ebb48ffcdce3" becomes "3.86.2-7".
func trimBuildMetadata(version string) string {
	trimmed, _, _ := strings.Cut(version, "+")
	return trimmed
}
