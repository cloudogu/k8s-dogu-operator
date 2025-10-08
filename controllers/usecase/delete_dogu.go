package usecase

import (
	"context"
	"fmt"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	client client.Client
	steps  []Step
}

func NewDoguDeleteUseCase(
	k8sClient client.Client,
	statusStep *deletion.StatusStep,
	serviceAccountRemoverStep *deletion.ServiceAccountRemoverStep,
	deleteOutOfHealthConfigMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeSensitiveDoguConfigStep deletion.RemoveSensitiveDoguConfigStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{
		client: k8sClient,
		steps: []Step{
			statusStep,
			serviceAccountRemoverStep,
			deleteOutOfHealthConfigMapStep,
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
			if result.Err != nil {
				if cloudoguerrors.IsNotFoundError(result.Err) {
					return 0, true, nil
				}
				logger.Error(result.Err, fmt.Sprintf("reconcile step %T has to requeue: %q", s, result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile step %T has to requeue after %d", s, result.RequeueAfter))
			}
			return result.RequeueAfter, false, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	logger.Info(fmt.Sprintf("successfully deleted dogu: %s", doguResource.Name))
	return 0, true, nil
}
