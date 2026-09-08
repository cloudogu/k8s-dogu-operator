package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	meta2 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WaitForOCIRepositoryReadyStep struct {
	k8sClient client.Client
}

func NewWaitForOCIRepositoryReadyStep(k8sClient client.Client) *WaitForOCIRepositoryReadyStep {
	return &WaitForOCIRepositoryReadyStep{k8sClient: k8sClient}
}

func (wor *WaitForOCIRepositoryReadyStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	repo := &flux.OCIRepository{}
	err := wor.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, repo)
	if err != nil {
		return stepsv3.StepResult{Err: fmt.Errorf("failed to get OCIRepository: %w", err)}
	}

	if isOCIRepositoryReady(repo) {
		// Status did not change, continue
		if !meta2.SetStatusCondition(&doguResource.Status.Conditions, getSuccessfulChartAvailableCondition(doguResource)) {
			return stepsv3.Continue()
		}

		updateErr := wor.k8sClient.Status().Update(ctx, doguResource)
		if updateErr != nil {
			return stepsv3.StepResult{Err: fmt.Errorf("failed to update dogu resource: %w", updateErr)}
		}
		return stepsv3.Continue()
	}

	// Not ready yet, requeue
	return stepsv3.RequeueAfter(5 * time.Second)
}

func getOCIRepositoryNotReadyReason(repo *flux.OCIRepository) string {
	for _, cond := range repo.Status.Conditions {
		if (cond.Type == flux.FetchFailedCondition || cond.Type == flux.IncludeUnavailableCondition || cond.Type == flux.StorageOperationFailedCondition) && cond.Status == metav1.ConditionTrue {
			switch cond.Reason {
			case flux.AuthenticationFailedReason:
				return v3beta1.ReasonUnauthorized
			case flux.
			}
		}
	}
	return false
}

func isOCIRepositoryReady(repo *flux.OCIRepository) bool {
	for _, cond := range repo.Status.Conditions {
		if cond.Type == meta.ReadyCondition && cond.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func getSuccessfulChartAvailableCondition(doguResource *v3beta1.Dogu) metav1.Condition {
	return getChartAvailableCondition(metav1.ConditionTrue, "", doguResource.Generation)
}

func getFailureChartAvailableCondition(doguResource *v3beta1.Dogu) metav1.Condition {
	return getChartAvailableCondition(metav1.ConditionFalse, "", doguResource.Generation)
}

func getChartAvailableCondition(status metav1.ConditionStatus, reason string, observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               v3beta1.ConditionChartAvailable,
		Status:             status,
		ObservedGeneration: observedGeneration,
		// TODO: set LastTransitionTime
		LastTransitionTime: metav1.Time{},
		Reason:             reason,
	}
}

/*// Reasons for ConditionChartAvailable.
ReasonChartNotFound  = "ChartNotFound"
ReasonDownloadFailed = "DownloadFailed"
ReasonUnauthorized   = "Unauthorized"*/
