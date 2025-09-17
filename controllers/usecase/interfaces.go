package usecase

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type Step interface {
	Run(ctx context.Context, resource *v2.Dogu) steps.StepResult
	// Priority for execution of the step. Higher priority means earlier execution.
	Priority() int
}
