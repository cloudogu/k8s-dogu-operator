package install

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterNetworkPoliciesStep = 10 * time.Second

type NetworkPoliciesStep struct {
	netPolUpserter   netPolUpserter
	localDoguFetcher localDoguFetcher
}

func NewNetworkPoliciesStep(mgrSet util.ManagerSet) *NetworkPoliciesStep {
	return &NetworkPoliciesStep{netPolUpserter: mgrSet.ResourceUpserter}
}

func (nps *NetworkPoliciesStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := nps.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor"))
	}
	err = nps.netPolUpserter.UpsertDoguNetworkPolicies(ctx, doguResource, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to setup network policies for dogu %s: %w", doguResource.Name, err))
	}
	return steps.StepResult{}
}
