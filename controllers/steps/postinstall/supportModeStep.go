package postinstall

import (
	"context"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const SupportModeEnvVar = "SUPPORT_MODE"

const (
	ReasonSupportModeActive   = "SupportModeActive"
	ReasonSupportModeInactive = "SupportModeInactive"
)

type SupportModeStep struct {
	supportManager supportManager
	client         client.Client
	doguInterface  doguInterface
}

func NewSupportModeStep(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder, namespace string) *SupportModeStep {
	return &SupportModeStep{
		supportManager: manager.NewDoguSupportManager(client, mgrSet, eventRecorder),
		client:         client,
		doguInterface:  mgrSet.EcosystemClient.Dogus(namespace),
	}
}

func (sms *SupportModeStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	_, err := sms.supportManager.HandleSupportMode(ctx, doguResource)
	deployment := &appsv1.Deployment{}
	err = sms.client.Get(ctx, doguResource.GetObjectKey(), deployment)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %s: %w", doguResource.Name, err))
	}

	doguResource, err = sms.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if isDeploymentInSupportMode(deployment) {
		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionTrue, ReasonSupportModeActive, "The Support mode is active")
		return steps.Abort()
	} else {
		err = sms.setSupportModeCondition(ctx, doguResource, metav1.ConditionFalse, ReasonSupportModeInactive, "The Support mode is inactive")
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
	logger := log.FromContext(ctx)
	condition := metav1.Condition{
		Type:               doguv2.ConditionSupportMode,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err := sms.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to update dogu resource"))
		return err
	}
	logger.Info(fmt.Sprintf("Updated dogu resource successfully!"))
	return nil
}
