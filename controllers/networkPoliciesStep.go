package controllers

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterNetworkPoliciesStep = 10 * time.Second

type networkPoliciesStep struct {
	netPolUpserter      netPolUpserter
	resourceDoguFetcher resourceDoguFetcher
}

func NewNetworkPoliciesStep(mgrSet util.ManagerSet) *networkPoliciesStep {
	return &networkPoliciesStep{netPolUpserter: mgrSet.ResourceUpserter}
}

func (nps *networkPoliciesStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	dogu, _, err := nps.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return requeueAfterNetworkPoliciesStep, fmt.Errorf("failed to fetch dogu descriptor")
	}
	err = nps.netPolUpserter.UpsertDoguNetworkPolicies(ctx, doguResource, dogu)
	if err != nil {
		return requeueAfterNetworkPoliciesStep, fmt.Errorf("failed to setup network policies for dogu %s: %w", doguResource.Name, err)
	}
	return 0, nil
}
