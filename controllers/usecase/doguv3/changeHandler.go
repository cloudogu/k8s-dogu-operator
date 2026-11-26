package doguv3

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/upgrade"
)

// DoguChangeHandler decides from the state of the cluster whether a dogu is installed or changed
// the regular way, has to be migrated from v2 or changes its version, and hands it to the matching use-case.
type DoguChangeHandler struct {
	migrationDetector MigrationDetector
	upgradeDetector   UpgradeDetector
	installUseCase    *DoguUseCase
	migrationUseCase  *DoguUseCase
	upgradeUseCase    *DoguUseCase
}

func NewDoguChangeHandler(migrationDetector MigrationDetector, upgradeDetector UpgradeDetector, installUseCase, migrationUseCase, upgradeUseCase *DoguUseCase) *DoguChangeHandler {
	return &DoguChangeHandler{
		migrationDetector: migrationDetector,
		upgradeDetector:   upgradeDetector,
		installUseCase:    installUseCase,
		migrationUseCase:  migrationUseCase,
		upgradeUseCase:    upgradeUseCase,
	}
}

func (r *DoguChangeHandler) HandleUntilApplied(ctx context.Context, doguResource *v3beta1.Dogu) (time.Duration, error) {
	state, err := r.migrationDetector.Detect(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to detect migration state of dogu %q: %w", doguResource.Name, err)
	}

	if state != migration.NotRequired {
		return r.migrationUseCase.HandleUntilApplied(ctx, doguResource)
	}

	change, err := r.upgradeDetector.Detect(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to detect version change of dogu %q: %w", doguResource.Name, err)
	}

	// Downgrades and invalid versions are handled by the upgrade use-case as well, which refuses them.
	if change.Kind != upgrade.NoChange {
		return r.upgradeUseCase.HandleUntilApplied(ctx, doguResource)
	}

	return r.installUseCase.HandleUntilApplied(ctx, doguResource)
}
