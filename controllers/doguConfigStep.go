package controllers

import (
	"context"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/config"
)

const requeueAfterDoguConfig = 5 * time.Second

type DoguConfigStep struct {
	doguConfigRepository doguConfigRepository
}

func NewDoguConfigStep(configRepos util.ConfigRepositories) *DoguConfigStep {
	return &DoguConfigStep{
		doguConfigRepository: configRepos.DoguConfigRepository,
	}
}

func (dcs *DoguConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	_, err = dcs.doguConfigRepository.Get(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		if !errors.IsNotFoundError(err) {
			return requeueAfterDoguConfig, err
		}

		err = dcs.createConfig(ctx, doguResource)
		if err != nil {
			return requeueAfterDoguConfig, err
		}
	}

	return 0, nil
}

func (dcs *DoguConfigStep) createConfig(ctx context.Context, doguResource *v2.Dogu) error {
	emptyCfg := config.CreateDoguConfig(cescommons.SimpleName(doguResource.Name), make(config.Entries))

	_, err := dcs.doguConfigRepository.Create(ctx, emptyCfg)
	if err != nil {
		return err
	}
	return nil
}
