package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstalledVersionStep struct {
	doguInterface doguInterface
}

func NewInstalledVersionStep(mgrSet *util.ManagerSet, namespace string) *InstalledVersionStep {
	return &InstalledVersionStep{
		doguInterface: mgrSet.EcosystemClient.Dogus(namespace),
	}
}

func (ivs *InstalledVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguResource, err := ivs.doguInterface.Get(ctx, doguResource.Name, v1.GetOptions{})
	if err != nil {
		steps.RequeueWithError(err)
	}
	doguResource.Status.InstalledVersion = doguResource.Spec.Version
	doguResource.Status.Status = v2.DoguStatusInstalled
	doguResource, err = ivs.doguInterface.UpdateStatus(ctx, doguResource, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
