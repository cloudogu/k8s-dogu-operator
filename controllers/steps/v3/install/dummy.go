package install

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DummyStep should be deleted.
type DummyStep struct {
	k8sClient client.Client
}

func NewDummyStep(k8sClient client.Client) *DummyStep {
	return &DummyStep{
		k8sClient: k8sClient,
	}
}

func (ds *DummyStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	// do something with generic k8sClient
	// err := ds.k8sClient.Update(ctx, doguResource)

	return stepsv3.Continue()
}
