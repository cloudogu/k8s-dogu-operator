package usecase

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type step interface {
	Run(ctx context.Context, resource *v2.Dogu) steps.StepResult
}

type doguRestartManager interface {
	RestartAllDogus(ctx context.Context) error
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}
