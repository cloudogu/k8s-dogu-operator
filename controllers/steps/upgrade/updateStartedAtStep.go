package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The UpdateStartedAtStep sets the startedAt time in the dogu status.
// The started at time is needed for the blueprint-operator.
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

	doguResource, err = usas.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus { //nolint:staticcheck
		status.StartedAt = metav1.Time{Time: *startingTime}
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
