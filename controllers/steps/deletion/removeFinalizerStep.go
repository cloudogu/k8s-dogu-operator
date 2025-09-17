package deletion

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const legacyFinalizerName = "dogu-finalizer"
const finalizerName = "k8s.cloudogu.com/dogu-cleanup"

type RemoveFinalizerStep struct {
	client k8sClient
}

func NewRemoveFinalizerStep(client client.Client) *RemoveFinalizerStep {
	return &RemoveFinalizerStep{
		client: client,
	}
}

func (rf *RemoveFinalizerStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	controllerutil.RemoveFinalizer(doguResource, legacyFinalizerName)
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err := rf.client.Update(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to update dogu: %w", err))
	}
	return steps.Continue()
}
