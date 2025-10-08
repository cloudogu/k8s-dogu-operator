package install

import (
	"context"
	"fmt"
	"github.com/cloudogu/retry-lib/retry"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizerName = "k8s.cloudogu.com/dogu-cleanup"

type FinalizerExistsStep struct {
	client k8sClient
}

func NewFinalizerExistsStep(client client.Client) *FinalizerExistsStep {
	return &FinalizerExistsStep{
		client: client,
	}
}

func (fs *FinalizerExistsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if controllerutil.ContainsFinalizer(doguResource, finalizerName) {
		return steps.Continue()
	}

	err := retry.OnConflict(func() error {
		err := fs.client.Get(ctx, client.ObjectKeyFromObject(doguResource), doguResource)
		if err != nil {
			return err
		}
		controllerutil.AddFinalizer(doguResource, finalizerName)
		return fs.client.Update(ctx, doguResource)
	})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to update dogu: %w", err))
	}

	return steps.Continue()
}
