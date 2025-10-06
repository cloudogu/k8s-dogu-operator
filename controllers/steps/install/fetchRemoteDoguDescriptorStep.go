package install

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dogu2 "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FetchRemoteDoguDescriptorStep struct {
	client                  k8sClient
	resourceDoguFetcher     resourceDoguFetcher
	localDoguDescriptorRepo localDoguDescriptorRepository
}

func NewFetchRemoteDoguDescriptorStep(client client.Client, localDoguDescriptorRepo dogu2.LocalDoguDescriptorRepository, resourceDoguFetcher cesregistry.ResourceDoguFetcher) *FetchRemoteDoguDescriptorStep {
	return &FetchRemoteDoguDescriptorStep{client: client, localDoguDescriptorRepo: localDoguDescriptorRepo, resourceDoguFetcher: resourceDoguFetcher}
}

func (f *FetchRemoteDoguDescriptorStep) Run(ctx context.Context, resource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("fetchRemoteDoguDescriptorStep")

	version, err := resource.GetSimpleNameVersion()
	if err != nil {
		return steps.RequeueWithError(err)
	}

	doguDescriptor, err := f.localDoguDescriptorRepo.Get(ctx, version)
	if err != nil && !cloudoguerrors.IsNotFoundError(err) {
		return steps.RequeueWithError(err)
	} else if err == nil && doguDescriptor != nil {
		return steps.Continue()
	}

	doguDescriptor, developmentDoguMap, err := f.resourceDoguFetcher.FetchWithResource(ctx, resource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = f.localDoguDescriptorRepo.Add(ctx, resource.GetSimpleDoguName(), doguDescriptor)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, f.client)
		if err != nil {
			logger.Error(err, "failed to delete development dogu map from cluster")
			return steps.Continue()
		}
	}

	return steps.Continue()
}
