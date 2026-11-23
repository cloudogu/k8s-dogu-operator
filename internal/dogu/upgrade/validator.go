package upgrade

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/dogu-lib/doguv3"
	doguv3reg "github.com/cloudogu/dogu-lib/doguv3/doguregistry"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

// ErrDescriptorUnavailable means the dogu registry temporarily fails to provide the upgrade descriptor.
var ErrDescriptorUnavailable = errors.New("upgrade descriptor unavailable")

// Verdict is the outcome of the validation of a version change.
type Verdict int

const (
	// Unchanged means the requested version is installed or the dogu is not installed yet, so there is nothing to validate.
	Unchanged Verdict = iota
	// Valid means the upgrade is allowed.
	Valid
	// Skipped means the upgrade is not validated against an upgrade descriptor, because the dogu uses a dev chart.
	Skipped
	// Refused means the version change is not allowed, see Cause.
	Refused
)

// Cause names why a version change is Refused.
type Cause int

const (
	// NoCause is the Cause of every Verdict but Refused.
	NoCause Cause = iota
	// CauseDowngrade means an older version than the installed one is requested.
	CauseDowngrade
	// CauseInvalidVersion means the requested or the installed version is not a valid semantic version.
	CauseInvalidVersion
	// CauseInvalidUpgradePath means the upgrade descriptor does not allow the upgrade.
	CauseInvalidUpgradePath
	// CauseInvalidDescriptor means the upgrade descriptor cannot be parsed or is ambiguous.
	CauseInvalidDescriptor
	// CauseDescriptorUnavailable means the dogu registry does not know or denies the target version.
	CauseDescriptorUnavailable
)

// Result is the validated version change of a dogu.
type Result struct {
	Change  Change
	Verdict Verdict
	// Cause names why the version change is refused. It is NoCause for every other Verdict.
	Cause Cause
	// Message describes the Verdict.
	Message string
}

// Validator decides whether the version change of a dogu is allowed.
type Validator struct {
	detector     ChangeDetector
	doguRegistry DoguRegistryReader
}

// NewValidator creates a Validator.
func NewValidator(detector ChangeDetector, doguRegistry DoguRegistryReader) *Validator {
	return &Validator{detector: detector, doguRegistry: doguRegistry}
}

// Validate validates the version change of the dogu against the upgrade descriptor of the target version from the dogu registry.
// Downgrades and invalid versions are refused. A dev chart does not come from the dogu registry, so its upgrade is Skipped.
//
// An error means the validation failed temporarily and should be retried.
// It wraps ErrDescriptorUnavailable if the dogu registry failed.
func (v *Validator) Validate(ctx context.Context, dogu *v3beta1.Dogu, devChart bool) (Result, error) {
	change, err := v.detector.Detect(ctx, dogu)
	if err != nil {
		return Result{}, fmt.Errorf("failed to detect version change of dogu %q: %w", dogu.Name, err)
	}

	switch change.Kind {
	case NoChange:
		return Result{
			Change:  change,
			Verdict: Unchanged,
		}, nil

	case Downgrade:
		return Result{
			Change:  change,
			Verdict: Refused,
			Cause:   CauseDowngrade,
			Message: fmt.Sprintf("Downgrade from version %s to %s is not allowed", change.Installed, change.Target),
		}, nil

	case InvalidVersion:
		return Result{
			Change:  change,
			Verdict: Refused,
			Cause:   CauseInvalidVersion,
			Message: fmt.Sprintf("Version change from %s to %s is not allowed because a version is not a valid semantic version", change.Installed, change.Target),
		}, err

	default:
	}

	upgradeName := fmt.Sprintf("Upgrade from version %s to %s", change.Installed, change.Target)
	if devChart {
		return Result{Change: change, Verdict: Skipped, Message: fmt.Sprintf("%s is not validated because the dogu uses a dev chart", upgradeName)}, nil
	}

	identifier := doguv3.Identifier{DoguNamespace: dogu.Spec.DoguNamespace, Name: dogu.Spec.Name, Version: change.Target}
	descriptor, err := v.doguRegistry.Get(ctx, identifier)
	if err != nil {
		return handleRegistryError(change, identifier, err)
	}

	if err = ValidatePath(change.Installed, change.Target, descriptor.Upgrades); err != nil {
		cause := CauseInvalidUpgradePath
		if errors.Is(err, ErrInvalidDescriptor) {
			cause = CauseInvalidDescriptor
		}

		return Result{
			Change:  change,
			Verdict: Refused,
			Cause:   cause,
			Message: fmt.Sprintf("%s is not allowed: %s", upgradeName, err),
		}, nil
	}

	return Result{Change: change, Verdict: Valid, Message: fmt.Sprintf("%s is valid", upgradeName)}, nil
}

func handleRegistryError(change Change, identifier doguv3.Identifier, err error) (Result, error) {
	wErr := fmt.Errorf("failed to get upgrade descriptor for identifier %s: %w", identifier, err)

	if doguv3reg.IsNotFoundError(err) || doguv3reg.IsUnauthorizedError(err) || doguv3reg.IsForbiddenError(err) {
		return Result{
			Change:  change,
			Verdict: Refused,
			Cause:   CauseDescriptorUnavailable,
			Message: wErr.Error(),
		}, nil
	}

	return Result{}, errors.Join(ErrDescriptorUnavailable, wErr)
}
