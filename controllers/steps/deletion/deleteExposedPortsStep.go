package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposedport"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// The DeleteOutOfHealthConfigMapStep remove the dogu out of the `k8s-dogu-operator-dogu-health` config map.
type DeleteExposedPortsStep struct {
	localDoguFetcher    localDoguFetcher
	exposedPortsManager exposedPortsManager
}

func NewDeleteExposedPortsStep(localDoguFetcher cesregistry.LocalDoguFetcher, mapInterface v1.ConfigMapInterface) *DeleteExposedPortsStep {
	return &DeleteExposedPortsStep{
		localDoguFetcher:    localDoguFetcher,
		exposedPortsManager: exposedport.NewExposedPortsManager(mapInterface),
	}
}

func (eps *DeleteExposedPortsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := eps.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = eps.exposedPortsManager.DeletePorts(ctx, dogu.ExposedPorts)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
