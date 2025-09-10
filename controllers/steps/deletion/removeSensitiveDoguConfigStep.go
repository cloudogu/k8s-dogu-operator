package deletion

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	registryErrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type RemoveSensitiveDoguConfigStep struct {
	sensitiveDoguRepository doguConfigRepository
}

func NewRemoveSensitiveDoguConfigStep(doguConfigRepository doguConfigRepository) *RemoveSensitiveDoguConfigStep {
	return &RemoveSensitiveDoguConfigStep{
		sensitiveDoguRepository: doguConfigRepository,
	}
}

func (rdc *RemoveSensitiveDoguConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := rdc.sensitiveDoguRepository.Delete(ctx, cescommons.SimpleName(doguResource.Name)); err != nil && !registryErrors.IsNotFoundError(err) {
		return steps.RequeueWithError(fmt.Errorf("could not delete snesitive dogu config: %w", err))
	}
	return steps.Continue()
}
