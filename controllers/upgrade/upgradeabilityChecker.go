package upgrade

import (
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
)

type upgradeabilityChecker struct {
}

// NewUpgradeabilityChecker creates a dogu upgradeability checker.
func NewUpgradeabilityChecker() *upgradeabilityChecker {
	return &upgradeabilityChecker{}
}

func (u *upgradeabilityChecker) Check(fromDogu *core.Dogu, toDogu *core.Dogu, forceUpgrade bool) error {
	if forceUpgrade {
		return nil
	}

	err := checkVersionBeforeUpgrade(fromDogu, toDogu)
	if err != nil {
		return fmt.Errorf("upgradeability check failed: %w", err)
	}

	return nil
}

func checkVersionBeforeUpgrade(fromDogu *core.Dogu, toDogu *core.Dogu) error {
	localVersion, err := core.ParseVersion(fromDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgradeability of local dogu: %w", err)
	}
	remoteVersion, err := core.ParseVersion(toDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgradeability of remote dogu: %w", err)
	}

	if remoteVersion.IsOlderOrEqualThan(localVersion) {
		return fmt.Errorf("remote version must be greater than local version '%s > %s'",
			toDogu.Version, fromDogu.Version)
	}
	return nil
}
