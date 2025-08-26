package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterHealthStep = 2 * time.Second

type HealthStep struct {
	resourceDoguFetcher        resourceDoguFetcher
	localDoguFetcher           localDoguFetcher
	dependencyValidator        DependencyValidator
	doguHealthChecker          doguHealthChecker
	doguRecursiveHealthChecker doguRecursiveHealthChecker
}

func NewHealthStep(mgrSet *util.ManagerSet) *HealthStep {
	doguChecker := health.NewDoguChecker(mgrSet.EcosystemClient, mgrSet.LocalDoguFetcher)
	return &HealthStep{
		resourceDoguFetcher:        mgrSet.ResourceDoguFetcher,
		localDoguFetcher:           mgrSet.LocalDoguFetcher,
		dependencyValidator:        mgrSet.DependencyValidator,
		doguHealthChecker:          doguChecker,
		doguRecursiveHealthChecker: doguChecker,
	}
}

func (hs *HealthStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	localDogu, err := hs.getLocalDogu(ctx, doguResource)
	if err != nil {
		return steps.StepResult{}
	}
	err = hs.doguHealthChecker.CheckByName(ctx, doguResource.GetObjectKey())
	if err != nil {
		return steps.StepResult{
			RequeueAfter: requeueAfterHealthStep,
		}
	}

	err = hs.checkDependencyDogusHealthy(ctx, localDogu, doguResource.Namespace)
	if err != nil {
		return steps.StepResult{
			RequeueAfter: requeueAfterHealthStep,
		}
	}
	return steps.StepResult{}
}

func (hs *HealthStep) getLocalDogu(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	dogu, err := hs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return nil, fmt.Errorf("dogu not found in local registry: %w", err)
	}

	return dogu, nil
}

func (hs *HealthStep) checkDoguIdentity(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	if localDogu.GetSimpleName() != remoteDogu.GetSimpleName() {
		return fmt.Errorf("dogus must have the same name (%s=%s)", localDogu.GetSimpleName(), remoteDogu.GetSimpleName())
	}

	if !namespaceChange && localDogu.GetNamespace() != remoteDogu.GetNamespace() {
		return fmt.Errorf("dogus must have the same namespace (%s=%s)", localDogu.GetNamespace(), remoteDogu.GetNamespace())
	}

	return nil
}
func (hs *HealthStep) checkDependencyDogusHealthy(
	ctx context.Context,
	localDogu *core.Dogu,
	namespace string,
) error {
	err := hs.dependencyValidator.ValidateDependencies(ctx, localDogu)
	if err != nil {
		return err
	}

	return hs.doguRecursiveHealthChecker.CheckDependenciesRecursive(ctx, localDogu, namespace)

}
