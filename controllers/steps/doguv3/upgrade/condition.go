package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// updateValidCondition sets ConditionValid of the dogu and persists it if it changed.
func updateValidCondition(ctx context.Context, k8sClient K8sClient, doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) error {
	changed := meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionValid,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})

	if !changed {
		return nil
	}

	if err := k8sClient.Status().Update(ctx, doguResource); err != nil {
		return fmt.Errorf("failed to update valid condition of dogu %q: %w", doguResource.Name, err)
	}

	return nil
}

// refuse invalidates the dogu and aborts the reconciliation, so the release of the dogu stays untouched.
func refuse(ctx context.Context, k8sClient K8sClient, doguResource *v3beta1.Dogu, reason, msg string) stepsv3.StepResult {
	if err := updateValidCondition(ctx, k8sClient, doguResource, metav1.ConditionFalse, reason, msg); err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonUpgrading)
	}

	logging.Info(ctx, msg, "dogu", doguResource.Name)

	return stepsv3.Abort(reason, msg)
}
