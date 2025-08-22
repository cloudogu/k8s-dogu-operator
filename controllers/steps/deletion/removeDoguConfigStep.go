package deletion

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	registryErrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type RemoveDoguConfigStep struct {
	doguConfigRepository doguConfigRepository
}

func NewRemoveDoguConfigStep(configRepos util.ConfigRepositories) *RemoveDoguConfigStep {
	return &RemoveDoguConfigStep{
		doguConfigRepository: configRepos.DoguConfigRepository,
	}
}

func (rdc *RemoveDoguConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := rdc.doguConfigRepository.Delete(ctx, cescommons.SimpleName(doguResource.Name)); err != nil && !registryErrors.IsNotFoundError(err) {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("could not delete dogu config: %w", err))
	}
	return steps.StepResult{}
}
