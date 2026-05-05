package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposedport"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type Values struct {
	Ports map[string]Port `yaml:"ports"`
}

type Port struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

type ExposePortStep struct {
	localDoguFetcher    localDoguFetcher
	exposedPortsManager exposedPortsManager
}

func NewExposePortStep(localDoguFetcher cesregistry.LocalDoguFetcher, mapInterface v1.ConfigMapInterface) *ExposePortStep {
	return &ExposePortStep{
		localDoguFetcher:    localDoguFetcher,
		exposedPortsManager: exposedport.NewExposedPortsManager(mapInterface),
	}
}

func (eps *ExposePortStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := eps.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = eps.exposedPortsManager.AddPorts(ctx, dogu.ExposedPorts)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
