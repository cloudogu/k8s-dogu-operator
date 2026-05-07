package install

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// The NetworkPoliciesStep creates or updates the Network Policies based on the dogu resource
type NetworkPoliciesStep struct {
	netPolUpserter   netPolUpserter
	localDoguFetcher localDoguFetcher
	imageRegistry    imageRegistry
	serviceInterface serviceInterface
}

func NewNetworkPoliciesStep(upserter resource.ResourceUpserter, fetcher cesregistry.LocalDoguFetcher, imageRegistry imageregistry.ImageRegistry, serviceInterface v1.ServiceInterface) *NetworkPoliciesStep {
	return &NetworkPoliciesStep{
		netPolUpserter:   upserter,
		localDoguFetcher: fetcher,
		imageRegistry:    imageRegistry,
		serviceInterface: serviceInterface,
	}
}

func (nps *NetworkPoliciesStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := nps.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	doguService, err := nps.serviceInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get dogu service for %q: %w", doguResource.Name, err))
	}

	imageConfig, err := nps.imageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to pull dogu image config for %q: %w", doguResource.Name, err))
	}

	routes, err := serviceaccess.CollectRoutes(doguService, &imageConfig.Config)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to collect dogu routes for %q: %w", doguResource.Name, err))
	}

	err = nps.netPolUpserter.UpsertDoguNetworkPolicies(ctx, doguResource, dogu, len(routes) > 0)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
