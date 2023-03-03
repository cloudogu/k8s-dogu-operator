package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type requeueablePremisesError struct {
	wrapped error
}

// Unwrap returns the next error in the error chain.
// If there is no next error, Unwrap returns nil.
func (r *requeueablePremisesError) Unwrap() error {
	return r.wrapped
}

// Error returns the string representation of an error.
func (r *requeueablePremisesError) Error() string {
	return r.wrapped.Error()
}

// Requeue is a interface marker method that always returns true when the error should produce a requeue for the
// current dogu resource operation.
func (r *requeueablePremisesError) Requeue() bool {
	return true
}

type premisesChecker struct {
	dependencyValidator        cloudogu.DependencyValidator
	doguHealthChecker          cloudogu.DoguHealthChecker
	doguRecursiveHealthChecker cloudogu.DoguRecursiveHealthChecker
}

// NewPremisesChecker creates a new upgrade premises checker.
func NewPremisesChecker(
	depValidator cloudogu.DependencyValidator,
	healthChecker cloudogu.DoguHealthChecker,
	recursiveHealthChecker cloudogu.DoguRecursiveHealthChecker,
) *premisesChecker {
	return &premisesChecker{
		dependencyValidator:        depValidator,
		doguHealthChecker:          healthChecker,
		doguRecursiveHealthChecker: recursiveHealthChecker,
	}
}

// Check tests if upgrade premises are valid and returns nil. Otherwise an error is returned to cancel the dogu upgrade
// early.
func (pc *premisesChecker) Check(
	ctx context.Context,
	doguResource *k8sv1.Dogu,
	localDogu *core.Dogu,
	remoteDogu *core.Dogu,
) error {
	changeNamespace := doguResource.Spec.UpgradeConfig.AllowNamespaceSwitch
	err := checkDoguIdentity(localDogu, remoteDogu, changeNamespace)
	// this error is most probably unrequeueable
	if err != nil {
		return err
	}

	err = pc.doguHealthChecker.CheckWithResource(ctx, doguResource)
	if err != nil {
		return &requeueablePremisesError{wrapped: err}
	}

	err = pc.checkDependencyDogusHealthy(ctx, doguResource, localDogu)
	if err != nil {
		return &requeueablePremisesError{wrapped: err}
	}

	return nil
}

func (pc *premisesChecker) checkDependencyDogusHealthy(
	ctx context.Context,
	doguResource *k8sv1.Dogu,
	localDogu *core.Dogu,
) error {
	err := pc.dependencyValidator.ValidateDependencies(ctx, localDogu)
	if err != nil {
		return err
	}

	return pc.doguRecursiveHealthChecker.CheckDependenciesRecursive(ctx, localDogu, doguResource.Namespace)

}

func checkDoguIdentity(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	if localDogu.GetSimpleName() != remoteDogu.GetSimpleName() {
		return fmt.Errorf("dogus must have the same name (%s=%s)", localDogu.GetSimpleName(), remoteDogu.GetSimpleName())
	}

	if !namespaceChange && localDogu.GetNamespace() != remoteDogu.GetNamespace() {
		return fmt.Errorf("dogus must have the same namespace (%s=%s)", localDogu.GetNamespace(), remoteDogu.GetNamespace())
	}

	return nil
}
