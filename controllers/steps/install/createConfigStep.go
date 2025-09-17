package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-registry-lib/config"
)

type CreateConfigStep struct {
	configRepository doguConfigRepository
}

func NewCreateConfigStep(configRepo initfx.DoguConfigRepository) *CreateConfigStep {
	return &CreateConfigStep{
		configRepository: configRepo,
	}
}

func (ccs *CreateConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := ccs.configRepository.Get(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		if !cloudoguerrors.IsNotFoundError(err) {
			return steps.RequeueWithError(err)
		}

		emptyCfg := config.CreateDoguConfig(cescommons.SimpleName(doguResource.Name), make(config.Entries))
		_, err = ccs.configRepository.Create(ctx, emptyCfg)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}
