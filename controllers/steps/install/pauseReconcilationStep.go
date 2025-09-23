package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type PauseReconcilationStep struct {
}

func NewPauseReconcilationStep() *PauseReconcilationStep {
	return &PauseReconcilationStep{}
}

func (prs *PauseReconcilationStep) Run(_ context.Context, doguResource *v2.Dogu) steps.StepResult {
	if doguResource.Spec.PauseReconcilation {
		return steps.Abort()
	}

	return steps.Continue()
}
