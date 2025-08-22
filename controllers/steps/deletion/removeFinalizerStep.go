package deletion

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizerName = "dogu-finalizer"

type RemoveFinalizerStep struct {
	client client.Client
}

func NewRemoveFinalizerStep(client client.Client) *RemoveFinalizerStep {
	return &RemoveFinalizerStep{
		client: client,
	}
}

func (rf *RemoveFinalizerStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err = rf.client.Update(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to update dogu: %w", err)
	}
	return 0, nil
}
