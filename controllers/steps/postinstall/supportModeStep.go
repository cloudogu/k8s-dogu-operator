package postinstall

import (
	"context"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

const SupportModeEnvVar = "SUPPORT_MODE"

const (
	ReasonSupportModeActive   = "SupportModeActive"
	ReasonSupportModeInactive = "SupportModeInactive"
)

// The SupportModeStep sets the dogu into support mode.
type SupportModeStep struct {
	supportManager      supportManager
	doguInterface       doguInterface
	deploymentInterface deploymentInterface
}

func NewSupportModeStep(supportManager manager.SupportManager, doguInterface doguClient.DoguInterface, deploymentInterface v1.DeploymentInterface) *SupportModeStep {
	return &SupportModeStep{
		supportManager:      supportManager,
		doguInterface:       doguInterface,
		deploymentInterface: deploymentInterface,
	}
}

func (sms *SupportModeStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	_, err := sms.supportManager.HandleSupportMode(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to handle support mode: %w", err))
	}

	deployment, err := sms.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %q: %w", doguResource.Name, err))
	}

	if isDeploymentInSupportMode(deployment) {
		doguResource.Status.Health = doguv2.UnavailableHealthStatus

		const message = "The Support mode is active"
		healthCondition := &metav1.Condition{
			Type:               doguv2.ConditionHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonSupportModeActive,
			Message:            message,
			ObservedGeneration: doguResource.Generation,
		}

		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionTrue, ReasonSupportModeActive, message, healthCondition)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		return steps.Abort()
	} else {
		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionFalse, ReasonSupportModeInactive, "The Support mode is inactive", nil)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		return steps.Continue()
	}
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

func (sms *SupportModeStep) setSupportModeCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string, healthCondition *metav1.Condition) error {
	condition := metav1.Condition{
		Type:               doguv2.ConditionSupportMode,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	}

	doguResource, err := sms.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		meta.SetStatusCondition(&status.Conditions, condition)
		if healthCondition != nil {
			meta.SetStatusCondition(&status.Conditions, *healthCondition)
		}
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
