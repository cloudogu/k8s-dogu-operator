package controllers

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguChangeUseCase struct {
	steps []step
}

func NewDoguChangeUseCase() *DoguChangeUseCase {
	return &DoguChangeUseCase{
		steps: []step{},
	}
}

func (dcu *DoguChangeUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration) {
	logger := log.FromContext(ctx).
		WithName("DoguChangeUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range dcu.steps {
		requeueAfter, err := s.Run(ctx, doguResource)
		if err != nil {
			logger.Error(err, "reconcile step failed: %w", err)
			return requeueAfter
		}
	}
	return
}
