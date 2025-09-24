package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstalledVersionStep struct {
	doguInterface doguInterface
}

func NewInstalledVersionStep(doguInterface doguClient.DoguInterface) *InstalledVersionStep {
	return &InstalledVersionStep{
		doguInterface: doguInterface,
	}
}

func (ivs *InstalledVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguResource.Status.InstalledVersion = doguResource.Spec.Version
	doguResource.Status.Status = v2.DoguStatusInstalled
	doguResource, err := ivs.doguInterface.UpdateStatus(ctx, doguResource, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
