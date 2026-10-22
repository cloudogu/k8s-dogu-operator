package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3beta1ac "github.com/cloudogu/k8s-dogu-lib/v3/applyconfiguration/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	conditionExportModeReason         = "NotImplemented"
	conditionOrEventExportModeMessage = "Export mode was activated but is currently not supported for V3 Dogus"
)

type ExportModeStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewExportModeStep(k8sClient K8sClient, recorder EventRecorder) *ExportModeStep {
	return &ExportModeStep{
		k8sClient:     k8sClient,
		eventRecorder: recorder,
	}
}

func getExportModeNotImplementedCondition(doguResource *v3beta1.Dogu) metav1.Condition {
	return metav1.Condition{
		Type:               v3beta1.ConditionExportModeActive,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: doguResource.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             conditionExportModeReason,
		Message:            conditionOrEventExportModeMessage,
	}
}

func (ems *ExportModeStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	exportMode := doguResource.Spec.ExportMode
	// just return if the condition is in the status or not and the export mode is false
	// we don't have to delete the warning condition.
	if !exportMode {
		return stepsv3.Continue()
	}

	// add condition if not set and export mode is active
	desiredCondition := getExportModeNotImplementedCondition(doguResource)
	condition := meta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionExportModeActive)
	if condition == nil {
		ems.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ConditionExportModeActive, conditionOrEventExportModeMessage)
		return ems.setCondition(ctx, doguResource, desiredCondition)
	}

	// Export mode is active and condition is set. We have to check the condition
	if conditionEquals(condition, desiredCondition) {
		return stepsv3.Continue()
	}

	// Update condition with correct values
	return ems.setCondition(ctx, doguResource, desiredCondition)
}

func conditionEquals(c1 *metav1.Condition, c2 metav1.Condition) bool {
	return c1.Status == c2.Status &&
		c1.Message == c2.Message &&
		c1.Reason == c2.Reason &&
		c1.ObservedGeneration == c2.ObservedGeneration
}

func (ems *ExportModeStep) setCondition(ctx context.Context, doguResource *v3beta1.Dogu, condition metav1.Condition) stepsv3.StepResult {
	conditionApplyConfiguration := v1.Condition().
		WithType(condition.Type).
		WithStatus(condition.Status).
		WithReason(condition.Reason).
		WithMessage(condition.Message).
		WithObservedGeneration(condition.ObservedGeneration).
		WithLastTransitionTime(condition.LastTransitionTime)
	doguApplyConfiguration := v3beta1ac.Dogu(doguResource.Name, doguResource.Namespace).
		WithStatus(v3beta1ac.DoguStatus().WithConditions(conditionApplyConfiguration))

	return ems.applyConfiguration(ctx, doguApplyConfiguration)
}

func (ems *ExportModeStep) applyConfiguration(ctx context.Context, configuration *v3beta1ac.DoguApplyConfiguration) stepsv3.StepResult {
	err := ems.k8sClient.Status().Apply(ctx, configuration, client.FieldOwner("k8s-dogu-operator-export-mode-step"), client.ForceOwnership)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to apply condition to status: %w", err), v3beta1.ReasonInstalling)
	}

	return stepsv3.Continue()
}
