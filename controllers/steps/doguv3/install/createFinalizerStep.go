package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateFinalizerStep ensures the dogu cr carries the cleanup finalizer so the operator can remove
// the resources it applied (OCIRepository, HelmRelease) before the dogu cr disappears.
type CreateFinalizerStep struct {
	client K8sClient
}

func NewCreateFinalizerStep(client K8sClient) *CreateFinalizerStep {
	return &CreateFinalizerStep{client: client}
}

func (fs *CreateFinalizerStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if controllerutil.ContainsFinalizer(doguResource, stepsv3.FinalizerName) {
		return stepsv3.Continue()
	}

	controllerutil.AddFinalizer(doguResource, stepsv3.FinalizerName)
	if err := fs.client.Update(ctx, doguResource); err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to add finalizer to dogu %q: %w", doguResource.Name, err), v3beta1.ReasonInstalling)
	}

	log.FromContext(ctx).Info("added cleanup finalizer to dogu", "name", doguResource.Name)
	return stepsv3.Continue()
}
