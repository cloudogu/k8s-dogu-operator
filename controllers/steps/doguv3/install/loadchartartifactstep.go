package install

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	flux "github.com/fluxcd/source-controller/api/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const chartLoadedMessage = "Chart artifact successfully downloaded, verified and loaded"

type LoadChartArtifactStep struct {
	chartLoader   ChartArtifactLoader
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewLoadChartArtifactStep(chartLoader ChartArtifactLoader, k8sClient K8sClient, recorder EventRecorder) *LoadChartArtifactStep {
	return &LoadChartArtifactStep{chartLoader: chartLoader, k8sClient: k8sClient, eventRecorder: recorder}
}

func (step *LoadChartArtifactStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	repository := &flux.OCIRepository{}
	err := step.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, repository)
	if apierrors.IsNotFound(err) {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, err.Error())
	}
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get OCIRepository for chart artifact: %w", err), v3beta1.ReasonInstalling)
	}
	if !isOCIRepositoryReady(repository) {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "waiting for current OCIRepository artifact")
	}
	if repository.Status.Artifact == nil || repository.Status.Artifact.URL == "" || repository.Status.Artifact.Digest == "" {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "waiting for OCIRepository artifact")
	}

	// GetChart validates and caches the chart for later installation steps.
	_, err = step.chartLoader.GetChart(ctx, repository.Status.Artifact.URL, repository.Status.Artifact.Digest)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to load OCIRepository artifact: %w", err)
		condition := getFailureChartAvailableCondition(doguResource, v3beta1.ReasonDownloadFailed, wrappedErr.Error())
		if errors.Is(err, errChartArtifactNotFound) {
			result := stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, wrappedErr.Error())
			return step.updateStatus(ctx, doguResource, condition, "", result)
		}
		if errors.Is(err, errInvalidChartArtifact) {
			result := stepsv3.Abort(v3beta1.ReasonDownloadFailed, wrappedErr.Error())
			return step.updateStatus(ctx, doguResource, condition, "", result)
		}

		result := stepsv3.RequeueWithError(wrappedErr, v3beta1.ReasonInstalling)
		return step.updateStatus(ctx, doguResource, condition, v1.EventTypeWarning, result)
	}

	condition := getChartAvailableCondition(metav1.ConditionTrue, v3beta1.ReasonSucceeded, chartLoadedMessage, doguResource.Generation)
	return step.updateStatus(ctx, doguResource, condition, v1.EventTypeNormal, stepsv3.Continue())
}

func (step *LoadChartArtifactStep) updateStatus(ctx context.Context, doguResource *v3beta1.Dogu, condition metav1.Condition, eventType string, result stepsv3.StepResult) stepsv3.StepResult {
	if !metautil.SetStatusCondition(&doguResource.Status.Conditions, condition) {
		return result
	}

	if stepResult, success := updateDoguResourceStatus(ctx, step.k8sClient, doguResource, "failed to update chart artifact condition"); !success {
		return stepResult
	}

	if eventType != "" {
		eventReason := v3beta1.ConditionChartAvailable
		if condition.Status == metav1.ConditionFalse {
			eventReason = v3beta1.ReasonDownloadFailed
		}
		step.eventRecorder.Event(doguResource, eventType, eventReason, condition.Message)
	}
	log.FromContext(ctx).Info(condition.Message)

	return result
}
