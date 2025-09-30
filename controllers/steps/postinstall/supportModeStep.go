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
		meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
			Type:               doguv2.ConditionHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonSupportModeActive,
			Message:            message,
			LastTransitionTime: steps.Now().Rfc3339Copy(),
		})

		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionTrue, ReasonSupportModeActive, message)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		return steps.Abort()
	} else {
		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionFalse, ReasonSupportModeInactive, "The Support mode is inactive")
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

func (sms *SupportModeStep) setSupportModeCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string) error {
	condition := metav1.Condition{
		Type:               doguv2.ConditionSupportMode,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: steps.Now().Rfc3339Copy(),
	}
	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err := sms.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
