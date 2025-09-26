package usecase

import (
	"context"
	"fmt"
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
	deleteOutOfHealthConfigMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{
		steps: []Step{
			serviceAccountRemoverStep,
			deleteOutOfHealthConfigMapStep,
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
			stepType := getType(s)
			if result.Err != nil {
				logger.Error(result.Err, fmt.Sprintf("reconcile step %s has to requeue: %q", stepType, result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile step %s has to requeue after %d", stepType, result.RequeueAfter))
			}
			return result.RequeueAfter, false, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	return 0, true, nil
}
