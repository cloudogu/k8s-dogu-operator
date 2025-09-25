package usecase

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	client client.Client
	steps  []Step
}

func NewDoguDeleteUseCase(
	k8sClient client.Client,
	serviceAccountRemoverStep *deletion.ServiceAccountRemoverStep,
	unregisterDoguVersionStep *deletion.UnregisterDoguVersionStep,
	deleteOutOfHealthConfigMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeDoguConfigStep deletion.RemoveDoguConfigStep,
	removeSensitiveDoguConfigStep deletion.RemoveSensitiveDoguConfigStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguDeleteUseCase {
	return &DoguDeleteUseCase{
		client: k8sClient,
		steps: []Step{
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
		err := ddu.client.Get(ctx, types.NamespacedName{Name: doguResource.Name, Namespace: doguResource.Namespace}, doguResource)
		if err != nil {
			return 0, false, err
		}

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
