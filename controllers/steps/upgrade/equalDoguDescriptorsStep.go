package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	semver "golang.org/x/mod/semver"
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
	remoteDescriptor, _, err := edds.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	localDescriptor, err := edds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if localDescriptor.Version == remoteDescriptor.Version {
		return steps.Continue()
	}

	if isOlder(remoteDescriptor.Version, localDescriptor.Version) {
		return steps.Abort()
	}

	err = edds.checkDoguIdentity(localDescriptor, remoteDescriptor, changeNamespace)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
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

func isOlder(version1, version2 string) bool {
	if version1[0] != 'v' {
		version1 = "v" + version1
	}
	if version2[0] != 'v' {
		version2 = "v" + version2
	}
	return semver.Compare(version1, version2) < 0
}
