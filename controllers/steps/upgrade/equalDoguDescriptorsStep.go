package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type EqualDoguDescriptorsStep struct {
	resourceDoguFetcher resourceDoguFetcher
	localDoguFetcher    localDoguFetcher
}

func NewEqualDoguDescriptorsStep(mgrSet *util.ManagerSet) *EqualDoguDescriptorsStep {
	return &EqualDoguDescriptorsStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
	}
}

func (edds *EqualDoguDescriptorsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	changeNamespace := doguResource.Spec.UpgradeConfig.AllowNamespaceSwitch
	remoteDescriptor, err := edds.getDoguDescriptor(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	localDescriptor, err := edds.getLocalDogu(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if localDescriptor.Version == remoteDescriptor.Version {
		return steps.Continue()
	}

	err = edds.checkDoguIdentity(localDescriptor, remoteDescriptor, changeNamespace)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}

func (edds *EqualDoguDescriptorsStep) getDoguDescriptor(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	doguDescriptor, _, err := edds.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return doguDescriptor, nil
}
func (edds *EqualDoguDescriptorsStep) getLocalDogu(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	dogu, err := edds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return nil, fmt.Errorf("dogu not found in local registry: %w", err)
	}

	return dogu, nil
}

func (edds *EqualDoguDescriptorsStep) checkDoguIdentity(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	if localDogu.GetSimpleName() != remoteDogu.GetSimpleName() {
		return fmt.Errorf("dogus must have the same name (%s=%s)", localDogu.GetSimpleName(), remoteDogu.GetSimpleName())
	}

	if !namespaceChange && localDogu.GetNamespace() != remoteDogu.GetNamespace() {
		return fmt.Errorf("dogus must have the same namespace (%s=%s)", localDogu.GetNamespace(), remoteDogu.GetNamespace())
	}

	return nil
}
