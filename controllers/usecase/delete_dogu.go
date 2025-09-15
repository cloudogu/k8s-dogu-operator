package usecase

import (
	"context"
	"slices"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	steps []Step
}

func NewDoguDeleteUseCase(steps []Step) *DoguDeleteUseCase {
	// sort descending because higher priority means earlier execution
	slices.SortFunc(steps, func(a, b Step) int {
		if a.Priority() < b.Priority() {
			return 1
		}
		if a.Priority() > b.Priority() {
			return -1
		}
		return 0
	})
	return &DoguDeleteUseCase{steps: steps}
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
