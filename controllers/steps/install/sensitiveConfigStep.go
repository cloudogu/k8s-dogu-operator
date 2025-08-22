package install

import (
	"context"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/config"
)

const requeueAfterSensitiveConfig = 5 * time.Second

type SensitiveConfigStep struct {
	sensitiveDoguRepository doguConfigRepository
}

func NewSensitiveConfigStep(configRepos util.ConfigRepositories) *SensitiveConfigStep {
	return &SensitiveConfigStep{
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
	}
}

func (scs *SensitiveConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	_, err = scs.sensitiveDoguRepository.Get(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		if !errors.IsNotFoundError(err) {
			return 0, err
		}

		err = scs.createConfig(ctx, doguResource)
		if err != nil {
			return 0, err
		}
	}

	return 0, nil
}

func (scs *SensitiveConfigStep) createConfig(ctx context.Context, doguResource *v2.Dogu) error {
	emptyCfg := config.CreateDoguConfig(cescommons.SimpleName(doguResource.Name), make(config.Entries))

	_, err := scs.sensitiveDoguRepository.Create(ctx, emptyCfg)
	if err != nil {
		return err
	}
	return nil
}
