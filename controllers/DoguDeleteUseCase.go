package controllers

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguDeleteUseCase struct {
	steps []step
}

func NewDoguDeleteUseCase(
	serviceAccountRemoverStep *ServiceAccountRemoverStep,
	unregisterVersionStep *UnregisterDoguVersionStep,
	healthMapStep *DeleteOutOfHealthConfigMapStep,
	removeDoguConfigStep *removeDoguConfigStep,
	removeSensitiveDoguConfigStep *removeSensitiveDoguConfigStep,
	removeFinalizerStep *removeFinalizerStep,
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
		requeueAfter, err := s.Run(ctx, doguResource)
		if err != nil || requeueAfter != 0 {
			logger.Error(err, "reconcile step has to requeue: %w", err)
			return requeueAfter, err
		}
	}
	return 0, nil
}
