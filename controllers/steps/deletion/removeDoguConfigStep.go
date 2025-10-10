package deletion

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	registryErrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The removeDoguConfigStep is used to delete the secret config of the dogu.
type removeDoguConfigStep struct {
	doguConfigRepository doguConfigRepository
}

func NewRemoveDoguConfigStep(doguConfigRepository initfx.DoguConfigRepository) *removeDoguConfigStep {
	return &removeDoguConfigStep{
		doguConfigRepository: doguConfigRepository,
	}
}

func (rdc *removeDoguConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := rdc.doguConfigRepository.Delete(ctx, cescommons.SimpleName(doguResource.Name)); err != nil && !registryErrors.IsNotFoundError(err) {
		return steps.RequeueWithError(fmt.Errorf("could not delete dogu config: %w", err))
	}
	return steps.Continue()
}
