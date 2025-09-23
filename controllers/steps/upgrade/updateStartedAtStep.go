package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type UpdateStartedAtStep struct {
	deploymentManager deploymentManager
	doguInterface     doguInterface
}

func NewUpdateStartedAtStep(
	doguInterface doguClient.DoguInterface,
	deploymentManager manager.DeploymentManager,
) *UpdateStartedAtStep {
	return &UpdateStartedAtStep{
		deploymentManager: deploymentManager,
		doguInterface:     doguInterface,
	}
}

func (usas *UpdateStartedAtStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	startingTime, err := usas.deploymentManager.GetLastStartingTime(ctx, doguResource.Name)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	doguResource.Status.StartedAt = metav1.Time{Time: *startingTime}
	doguResource, err = usas.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
