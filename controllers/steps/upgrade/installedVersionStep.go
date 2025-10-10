package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The InstalledVersionStep sets the currently installed version in the spec and sets the status to installed.
type InstalledVersionStep struct {
	doguInterface doguInterface
}

func NewInstalledVersionStep(doguInterface doguClient.DoguInterface) *InstalledVersionStep {
	return &InstalledVersionStep{
		doguInterface: doguInterface,
	}
}

func (ivs *InstalledVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	updatedDogu, err := ivs.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus {
		status.InstalledVersion = doguResource.Spec.Version
		status.Status = v2.DoguStatusInstalled
		return status
	}, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}
	*doguResource = *updatedDogu
	return steps.Continue()
}
