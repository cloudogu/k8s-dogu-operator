package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type NetworkPoliciesStep struct {
	netPolUpserter   netPolUpserter
	localDoguFetcher localDoguFetcher
}

func (nps *NetworkPoliciesStep) Priority() int {
	return 4100
}

func NewNetworkPoliciesStep(upserter resource.ResourceUpserter, fetcher cesregistry.LocalDoguFetcher) *NetworkPoliciesStep {
	return &NetworkPoliciesStep{
		netPolUpserter:   upserter,
		localDoguFetcher: fetcher,
	}
}

func (nps *NetworkPoliciesStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := nps.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = nps.netPolUpserter.UpsertDoguNetworkPolicies(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
