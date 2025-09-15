package usecase

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	Run(ctx context.Context, resource *v2.Dogu) steps.StepResult
	// Priority for execution of the step. Higher priority means earlier execution.
	Priority() int
}

type doguRestartManager interface {
	RestartAllDogus(ctx context.Context) error
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}

type k8sClient interface {
	client.Client
}
