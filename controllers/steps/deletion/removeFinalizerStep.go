package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/retry-lib/retry"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const legacyFinalizerName = "dogu-finalizer"
const finalizerName = "k8s.cloudogu.com/dogu-cleanup"

// The RemoveFinalizerStep removes the finalizer of the dogu resource.
type RemoveFinalizerStep struct {
	client k8sClient
}

func NewRemoveFinalizerStep(client client.Client) *RemoveFinalizerStep {
	return &RemoveFinalizerStep{
		client: client,
	}
}

func (rf *RemoveFinalizerStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if !controllerutil.ContainsFinalizer(doguResource, legacyFinalizerName) && !controllerutil.ContainsFinalizer(doguResource, finalizerName) {
		return steps.Continue()
	}

	err := retry.OnConflict(func() error {
		err := rf.client.Get(ctx, client.ObjectKeyFromObject(doguResource), doguResource)
		if err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(doguResource, legacyFinalizerName)
		controllerutil.RemoveFinalizer(doguResource, finalizerName)
		return rf.client.Update(ctx, doguResource)
	})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to update dogu: %w", err))
	}
	return steps.Continue()
}
