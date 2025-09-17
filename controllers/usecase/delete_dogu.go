package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	steps []Step
}

func NewDoguDeleteUseCase(
	serviceAccountRemoverStep *deletion.ServiceAccountRemoverStep,
	unregisterDoguVersionStep *deletion.UnregisterDoguVersionStep,
	deleteOutOfHealthConfigMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeDoguConfigStep deletion.RemoveDoguConfigStep,
	removeSensitiveDoguConfigStep deletion.RemoveSensitiveDoguConfigStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{steps: []Step{
		serviceAccountRemoverStep,
		unregisterDoguVersionStep,
		deleteOutOfHealthConfigMapStep,
		removeDoguConfigStep,
		removeSensitiveDoguConfigStep,
		removeFinalizerStep,
	}}
}

func (ddu *DoguDeleteUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error) {
	logger := log.FromContext(ctx).
		WithName("DoguDeleteUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range ddu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			logger.Error(result.Err, "reconcile Step has to requeue: %w", result.Err)
			return result.RequeueAfter, true, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	return 0, true, nil
}
