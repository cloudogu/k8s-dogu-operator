package usecase

import (
	"context"
	"fmt"
	"slices"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguInstallOrChangeUseCase struct {
	steps []Step
}

func NewDoguInstallOrChangeUseCase(steps []Step) *DoguInstallOrChangeUseCase {
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
	return &DoguInstallOrChangeUseCase{
		steps: steps,
	}
}

func (dicu *DoguInstallOrChangeUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error) {
	logger := log.FromContext(ctx).
		WithName("DoguChangeUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range dicu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			if result.Err != nil {
				logger.Error(result.Err, fmt.Sprintf("reconcile Step has to requeue: %q", result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile Step has to requeue after %d", result.RequeueAfter))
			}
			return result.RequeueAfter, true, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	logger.Info(fmt.Sprintf("Successfully went through all steps!"))
	return 0, true, nil
}
