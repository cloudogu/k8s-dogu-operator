package deletion

import (
	"context"
	"fmt"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	registryErrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type RemoveSensitiveDoguConfigStep struct {
	sensitiveDoguRepository doguConfigRepository
}

func NewRemoveSensitiveDoguConfigStep(configRepos util.ConfigRepositories) *RemoveSensitiveDoguConfigStep {
	return &RemoveSensitiveDoguConfigStep{
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
	}
}

func (rdc *RemoveSensitiveDoguConfigStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	if err = rdc.sensitiveDoguRepository.Delete(ctx, cescommons.SimpleName(doguResource.Name)); err != nil && !registryErrors.IsNotFoundError(err) {
		return 0, fmt.Errorf("could not delete snesitive dogu config: %w", err)
	}
	return 0, nil
}
