package controllers

import (
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
)

type upgradeChecker struct {
}

func (u *upgradeChecker) IsUpgradeable(fromDogu *core.Dogu, toDogu *core.Dogu, forceUpgrade bool) (bool, error) {
	if forceUpgrade {
		return true, nil
	}

	return u.checkVersionBeforeUpgrade(fromDogu, toDogu)
}

func (u *upgradeChecker) checkVersionBeforeUpgrade(fromDogu *core.Dogu, toDogu *core.Dogu) (bool, error) {
	localVersion, err := core.ParseVersion(fromDogu.Version)
	if err != nil {
		return false, fmt.Errorf("could not check upgradeability of local dogu: %w", err)
	}
	remoteVersion, err := core.ParseVersion(toDogu.Version)
	if err != nil {
		return false, fmt.Errorf("could not check upgradeability of remote dogu: %w", err)
	}

	if remoteVersion.IsOlderThan(localVersion) {
		return false, fmt.Errorf("downgrade from %s to %s is not allowed", fromDogu.Version, toDogu.Version)
	}

	return remoteVersion.IsNewerThan(localVersion), nil
}
