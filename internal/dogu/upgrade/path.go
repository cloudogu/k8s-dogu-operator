package upgrade

import (
	"errors"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/cloudogu/dogu-lib/doguv3"
)

var (
	// ErrNoUpgradePath means the upgrade descriptor does not allow the upgrade from the installed to the target version.
	ErrNoUpgradePath = errors.New("no upgrade path")
	// ErrInvalidDescriptor means the upgrade descriptor cannot be parsed or isn't explicit.
	ErrInvalidDescriptor = errors.New("invalid upgrade descriptor")
)

// pathStep is a parsed entry of the upgrade descriptor.
type pathStep struct {
	from *semver.Constraints
	to   *semver.Version
}

// ValidatePath checks whether the upgrade from the installed to the target version is allowed by the upgrade descriptor of the target version.
// A dogu without upgrade descriptor allows every upgrade.
//
// An upgrade must not span more than one entry of the descriptor: the target has to lie in the From range of the entry
// that contains the installed version or equal its To version. A target further along the path is refused with the
// To version as next upgrade step.
func ValidatePath(installed, target string, upgrades []doguv3.Upgrade) error {
	if len(upgrades) == 0 {
		return nil
	}

	installedVersion, err := semver.NewVersion(installed)
	if err != nil {
		return fmt.Errorf("failed to parse installed version %q: %w", installed, err)
	}

	targetVersion, err := semver.NewVersion(target)
	if err != nil {
		return fmt.Errorf("failed to parse target version %q: %w", target, err)
	}

	steps, err := parseSteps(upgrades)
	if err != nil {
		return fmt.Errorf("%w: failed to parse upgrade descriptor: %w", ErrInvalidDescriptor, err)
	}

	step, found, err := findStep(steps, installedVersion)
	if err != nil {
		return fmt.Errorf("%w: failed to find upgrade step: %w", ErrInvalidDescriptor, err)
	}

	if !found {
		return fmt.Errorf("%w: installed version %s is not part of the upgrade descriptor", ErrNoUpgradePath, installed)
	}

	if reaches(step, targetVersion) {
		return nil
	}

	reachable, err := reachableLater(steps, step.to, targetVersion)
	if err != nil {
		return err
	}

	if reachable {
		return fmt.Errorf("%w: target version %s requires more than one upgrade step; upgrade to %s first", ErrNoUpgradePath, target, step.to)
	}

	return fmt.Errorf("%w: target version %s cannot be reached from installed version %s", ErrNoUpgradePath, target, installed)
}

// reaches reports whether the step upgrades to the target version.
func reaches(step pathStep, target *semver.Version) bool {
	return step.from.Check(target) || step.to.Equal(target)
}

// reachableLater reports whether the target version can be reached by following the path from the given version.
// It only serves the hint of the next upgrade step, because an upgrade must not span several steps.
func reachableLater(steps []pathStep, from, target *semver.Version) (bool, error) {
	current := from
	// Every entry is followed at most once, so a cyclic descriptor cannot loop forever.
	for range steps {
		step, found, err := findStep(steps, current)
		if err != nil {
			return false, fmt.Errorf("%w: failed to find upgrade step: %w", ErrInvalidDescriptor, err)
		}

		if !found {
			return false, nil
		}

		if reaches(step, target) {
			return true, nil
		}

		current = step.to
	}

	return false, nil
}

func parseSteps(upgrades []doguv3.Upgrade) ([]pathStep, error) {
	steps := make([]pathStep, 0, len(upgrades))
	for _, upgrade := range upgrades {
		from, err := semver.NewConstraint(upgrade.From)
		if err != nil {
			return nil, fmt.Errorf("failed to parse from range %q: %w", upgrade.From, err)
		}

		to, err := semver.NewVersion(upgrade.To)
		if err != nil {
			return nil, fmt.Errorf("failed to parse to version %q: %w", upgrade.To, err)
		}

		steps = append(steps, pathStep{from: from, to: to})
	}

	return steps, nil
}

// findStep returns the entry whose From range contains the version. Several matching entries make the path ambiguous.
func findStep(steps []pathStep, version *semver.Version) (pathStep, bool, error) {
	var match pathStep
	found := false

	for _, step := range steps {
		if !step.from.Check(version) {
			continue
		}

		if found {
			return pathStep{}, false, fmt.Errorf("version %s matches the from ranges %q and %q", version, match.from, step.from)
		}

		match, found = step, true
	}

	return match, found, nil
}
