package doguv3

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
)

// DoguChangeHandler decides from the state of the cluster whether a dogu is installed or changed
// the regular way or has to be migrated from v2, and hands it to the matching use-case.
type DoguChangeHandler struct {
	detector         MigrationDetector
	installUseCase   *DoguUseCase
	migrationUseCase *DoguUseCase
}

func NewDoguChangeHandler(detector MigrationDetector, installUseCase, migrationUseCase *DoguUseCase) *DoguChangeHandler {
	return &DoguChangeHandler{detector: detector, installUseCase: installUseCase, migrationUseCase: migrationUseCase}
}

func (r *DoguChangeHandler) HandleUntilApplied(ctx context.Context, doguResource *v3beta1.Dogu) (time.Duration, error) {
	state, err := r.detector.Detect(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to detect migration state of dogu %q: %w", doguResource.Name, err)
	}

	if state != migration.NotRequired {
		return r.migrationUseCase.HandleUntilApplied(ctx, doguResource)
	}

	return r.installUseCase.HandleUntilApplied(ctx, doguResource)
}
