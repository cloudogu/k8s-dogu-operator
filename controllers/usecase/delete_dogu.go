package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	steps []step
}

func NewDoguDeleteUsecase(client k8sClient, mgrSet *util.ManagerSet, configRepos util.ConfigRepositories, operatorConfig *config.OperatorConfig) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{
		steps: []step{
			deletion.NewServiceAccountRemoverStep(client, mgrSet, configRepos, operatorConfig),
			deletion.NewUnregisterDoguVersionStep(mgrSet.DoguRegistrator),
			deletion.NewDeleteOutOfHealthConfigMapStep(client),
			deletion.NewRemoveDoguConfigStep(configRepos.DoguConfigRepository),
			deletion.NewRemoveSensitiveDoguConfigStep(configRepos.SensitiveDoguRepository),
			deletion.NewRemoveFinalizerStep(client),
		},
	}
}

func (ddu *DoguDeleteUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error) {
	logger := log.FromContext(ctx).
		WithName("DoguDeleteUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range ddu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			logger.Error(result.Err, "reconcile step has to requeue: %w", result.Err)
			return result.RequeueAfter, true, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	return 0, true, nil
}
