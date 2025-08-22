package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type step interface {
	Run(ctx context.Context, resource *v2.Dogu) (requeueAfter time.Duration, err error)
}

type doguUsecase interface {
	HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error)
}
