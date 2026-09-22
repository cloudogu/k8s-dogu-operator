package install

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultRequeueAfter = 5 * time.Second
	successMessage      = "Chart successfully downloaded by source-controller"
)

type WaitForOCIRepositoryReadyStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewWaitForOCIRepositoryReadyStep(k8sClient K8sClient, recorder EventRecorder) *WaitForOCIRepositoryReadyStep {
	return &WaitForOCIRepositoryReadyStep{k8sClient: k8sClient, eventRecorder: recorder}
}

// Run checks the ready condition of the previously created flux OCIRepository.
// If it is ready, the step continues, and if not, the step reconciles the dogu resource.
func (wor *WaitForOCIRepositoryReadyStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	repo := &flux.OCIRepository{}
	err := wor.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, repo)
	// cache lag
	if errors.IsNotFound(err) {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, err.Error())
	}
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get OCIRepository: %w", err), v3beta1.ReasonInstalling)
	}

	// Not ready yet, update status and requeue
	if !metautil.IsStatusConditionTrue(repo.Status.Conditions, meta.ReadyCondition) {
		return wor.updateChartUnavailableStatus(ctx, doguResource, repo)
	}

	// If success status did not change, continue with next step
	if !metautil.SetStatusCondition(&doguResource.Status.Conditions, getSuccessfulChartAvailableCondition(doguResource)) {
		return stepsv3.Continue()
	}

	if stepResult, success := wor.updateDoguResourceStatus(ctx, doguResource, "failed to update oci success condition"); success {
		wor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, successMessage)
		log.FromContext(ctx).Info(successMessage)
		return stepsv3.Continue()
	} else {
		return stepResult
	}
}

func (wor *WaitForOCIRepositoryReadyStep) updateChartUnavailableStatus(ctx context.Context, doguResource *v3beta1.Dogu, repo *flux.OCIRepository) stepsv3.StepResult {
	reason, msg, found := getOCIRepositoryChartAvailableReasonMessage(repo)
	// Not ready yet, requeue
	if !found {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "")
	}

	// Status did not change, abort
	// We abort here because 404 chart not found or 404 secret not found won't heal themselves.
	// If the secret is created or the correct Chart URL updated in the dogu cr, the operator will reconcile and heal.
	// Additionally, if the OCIRepository is updated by the source-controller the operator will reconcile too.
	if !metautil.SetStatusCondition(&doguResource.Status.Conditions, getFailureChartAvailableCondition(doguResource, reason, msg)) {
		return stepsv3.Abort(reason, msg)
	}

	// Write ChartUnavailable status to resource
	if stepResult, success := wor.updateDoguResourceStatus(ctx, doguResource, "failed to update oci failure condition"); !success {
		return stepResult
	}

	// Abort because see above
	return stepsv3.Abort(reason, msg)
}

func (wor *WaitForOCIRepositoryReadyStep) updateDoguResourceStatus(ctx context.Context, doguResource *v3beta1.Dogu, errorMessage string) (stepsv3.StepResult, bool) {
	updateErr := wor.k8sClient.Status().Update(ctx, doguResource)
	if updateErr != nil {
		// cache lag
		if errors.IsConflict(updateErr) {
			return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, updateErr.Error()), false
		}
		return stepsv3.RequeueWithError(fmt.Errorf("%s: %w", errorMessage, updateErr), v3beta1.DoguStatusInstalling), false
	}

	return stepsv3.StepResult{}, true
}

func getOCIRepositoryChartAvailableReasonMessage(repo *flux.OCIRepository) (string, string, bool) {
	fetchFailedCondition := metautil.FindStatusCondition(repo.Status.Conditions, flux.FetchFailedCondition)
	if fetchFailedCondition != nil && fetchFailedCondition.Status == metav1.ConditionTrue {
		switch fetchFailedCondition.Reason {
		case flux.AuthenticationFailedReason:
			return v3beta1.ReasonDownloadFailed, fetchFailedCondition.Message, true
		case flux.OCIPullFailedReason:
			msg := strings.ToLower(fetchFailedCondition.Message)
			if strings.Contains(msg, "unauthorized") {
				return v3beta1.ReasonUnauthorized, fetchFailedCondition.Message, true
			} else if strings.Contains(msg, "not found") {
				return v3beta1.ReasonChartNotFound, fetchFailedCondition.Message, true
			}
			return v3beta1.ReasonDownloadFailed, fetchFailedCondition.Message, true
		default:
			return v3beta1.ReasonDownloadFailed, fetchFailedCondition.Message, true
		}
	}

	// Use ready condition as fallback if no fetch failed condition is present
	readyCondition := metautil.FindStatusCondition(repo.Status.Conditions, meta.ReadyCondition)
	if readyCondition != nil && readyCondition.Status == metav1.ConditionFalse {
		return v3beta1.ReasonDownloadFailed, readyCondition.Message, true
	}

	return "", "", false
}

func getSuccessfulChartAvailableCondition(doguResource *v3beta1.Dogu) metav1.Condition {
	return getChartAvailableCondition(metav1.ConditionTrue, v3beta1.ReasonSucceeded, successMessage, doguResource.Generation)
}

func getFailureChartAvailableCondition(doguResource *v3beta1.Dogu, reason, msg string) metav1.Condition {
	return getChartAvailableCondition(metav1.ConditionFalse, reason, msg, doguResource.Generation)
}

func getChartAvailableCondition(status metav1.ConditionStatus, reason, msg string, observedGeneration int64) metav1.Condition {
	return metav1.Condition{
		Type:               v3beta1.ConditionChartAvailable,
		Status:             status,
		ObservedGeneration: observedGeneration,
		Reason:             reason,
		Message:            msg,
	}
}
