package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RemoveFinalizerStep removes the cleanup finalizer as the last deletion step, after the flux
// resources are gone. Once removed, Kubernetes completes the removal of the dogu cr.
type RemoveFinalizerStep struct {
	client K8sClient
}

func NewRemoveFinalizerStep(client K8sClient) *RemoveFinalizerStep {
	return &RemoveFinalizerStep{client: client}
}

func (rf *RemoveFinalizerStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if !controllerutil.ContainsFinalizer(doguResource, stepsv3.FinalizerName) {
		return stepsv3.Continue()
	}

	controllerutil.RemoveFinalizer(doguResource, stepsv3.FinalizerName)
	if uErr := rf.client.Update(ctx, doguResource); uErr != nil {
		// The dogu cr is already gone - nothing left to do.
		if apierrors.IsNotFound(uErr) {
			return stepsv3.Continue()
		}

		return stepsv3.RequeueWithError(fmt.Errorf("failed to remove finalizer from dogu %q: %w", doguResource.Name, uErr), stepsv3.ReasonDeletionFailed)
	}

	log.FromContext(ctx).Info("removed cleanup finalizer from dogu", "name", doguResource.Name)

	return stepsv3.Continue()
}
