package postinstall

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const SupportModeEnvVar = "SUPPORT_MODE"

type SupportModeStep struct {
	supportManager supportManager
	client         client.Client
}

func NewSupportModeStep(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *SupportModeStep {
	return &SupportModeStep{
		supportManager: manager.NewDoguSupportManager(client, mgrSet, eventRecorder),
		client:         client,
	}
}

func (sms *SupportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := sms.supportManager.HandleSupportMode(ctx, doguResource)
	deployment := &appsv1.Deployment{}
	err = sms.client.Get(ctx, doguResource.GetObjectKey(), deployment)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %s: %w", doguResource.Name, err))
	}

	return steps.StepResult{Continue: !isDeploymentInSupportMode(deployment)}
}

func isDeploymentInSupportMode(deployment *appsv1.Deployment) bool {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		envVars := container.Env
		for _, env := range envVars {
			if env.Name == SupportModeEnvVar && env.Value == "true" {
				return true
			}
		}
	}

	return false
}
