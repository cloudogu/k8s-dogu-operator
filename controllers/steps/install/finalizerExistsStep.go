package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizerName = "dogu-finalizer"

type FinalizerExistsStep struct{}

func NewFinalizerExistsStep() *FinalizerExistsStep {
	return &FinalizerExistsStep{}
}

func (fs *FinalizerExistsStep) Run(_ context.Context, doguResource *v2.Dogu) steps.StepResult {
	controllerutil.AddFinalizer(doguResource, finalizerName)
	return steps.Continue()
}
