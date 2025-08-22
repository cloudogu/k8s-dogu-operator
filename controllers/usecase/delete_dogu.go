package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	steps []step
}

func NewDoguDeleteUseCase(
	serviceAccountRemoverStep *deletion.ServiceAccountRemoverStep,
	unregisterVersionStep *deletion.UnregisterDoguVersionStep,
	healthMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeDoguConfigStep *deletion.RemoveDoguConfigStep,
	removeSensitiveDoguConfigStep *deletion.RemoveSensitiveDoguConfigStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{
		steps: []step{
			serviceAccountRemoverStep,
			unregisterVersionStep,
			healthMapStep,
			removeDoguConfigStep,
			removeSensitiveDoguConfigStep,
			removeFinalizerStep,
		},
	}
}

func (ddu *DoguDeleteUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, error) {
	logger := log.FromContext(ctx).
		WithName("DoguDeleteUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range ddu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			logger.Error(result.Err, "reconcile step has to requeue: %w", result.Err)
			return result.RequeueAfter, result.Err
		}
		if !result.Continue {
			break
		}
	}
	return 0, nil
}
