package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type step interface {
	Run(ctx context.Context, resource *v2.Dogu) steps.StepResult
}

type doguUsecase interface {
	HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, error)
}
