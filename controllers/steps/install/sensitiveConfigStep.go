package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/config"
)

type SensitiveConfigStep struct {
	sensitiveDoguRepository doguConfigRepository
}

func NewSensitiveConfigStep(configRepos util.ConfigRepositories) *SensitiveConfigStep {
	return &SensitiveConfigStep{
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
	}
}

func (scs *SensitiveConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := scs.sensitiveDoguRepository.Get(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		if !cloudoguerrors.IsNotFoundError(err) {
			return steps.RequeueWithError(err)
		}

		err = scs.createConfig(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}

func (scs *SensitiveConfigStep) createConfig(ctx context.Context, doguResource *v2.Dogu) error {
	emptyCfg := config.CreateDoguConfig(cescommons.SimpleName(doguResource.Name), make(config.Entries))

	_, err := scs.sensitiveDoguRepository.Create(ctx, emptyCfg)
	if err != nil {
		return err
	}
	return nil
}
