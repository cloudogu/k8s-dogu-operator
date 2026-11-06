package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
)

// MigrationVersionStep checks that the v3 chart contains the same application version as the installed v2 dogu.
type MigrationVersionStep struct {
	detector      MigrationDetector
	chartReader   ChartAppVersionReader
	eventRecorder EventRecorder
}

func NewMigrationVersionStep(detector MigrationDetector, chartReader ChartAppVersionReader, recorder EventRecorder) *MigrationVersionStep {
	return &MigrationVersionStep{detector: detector, chartReader: chartReader, eventRecorder: recorder}
}

func (s *MigrationVersionStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	state, err := s.detector.Detect(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to detect migration state of dogu %q: %w", doguResource.Name, err), v3beta1.ReasonMigrationPending)
	}
	if state != dogumigration.Required {
		return stepsv3.Continue()
	}

	// The status still holds the version the v2 reconciler wrote, e.g. 2.4.48-3.
	v2Version := doguResource.Status.InstalledVersion
	if v2Version == "" {
		return s.abort(doguResource, "The migration to v3 is blocked: the installed version of the v2 dogu is unknown")
	}

	appVersion, err := s.chartReader.AppVersion(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to read app version of the chart of dogu %q: %w", doguResource.Name, err), v3beta1.ReasonMigrationPending)
	}
	if appVersion == "" {
		return s.abort(doguResource, fmt.Sprintf("The migration to v3 is blocked: the chart version %q does not have an appVersion set", doguResource.Spec.Version))
	}

	installedAppVersion := withoutDoguRevision(v2Version)
	if installedAppVersion != appVersion {
		return s.abort(doguResource, fmt.Sprintf(
			"The migration to v3 is blocked: the appVersion %q does not match the version %q of the installed v2 dogu",
			appVersion, installedAppVersion))
	}

	return stepsv3.Continue()
}

// abort stops the reconciliation, because only a change of the dogu spec can resolve the mismatch.
func (s *MigrationVersionStep) abort(doguResource *v3beta1.Dogu, msg string) stepsv3.StepResult {
	s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationFailed, msg)
	return stepsv3.Abort(v3beta1.ReasonMigrationFailed, msg)
}

// withoutDoguRevision removes the dogu revision ("-N") from a v2 dogu version, e.g. 2.4.48-3 becomes 2.4.48.
func withoutDoguRevision(doguVersion string) string {
	version, _, _ := strings.Cut(doguVersion, "-")
	return version
}
