package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StopWorkloadsStep struct {
	k8sClient client.Client
	discovery VolumeDiscovery
	scaler    WorkloadScaler
}

func NewStopWorkloadsStep(k8sClient client.Client) *StopWorkloadsStep {
	return &StopWorkloadsStep{
		k8sClient: k8sClient,
		discovery: workloads.NewDiscovery(k8sClient),
		scaler:    workloads.NewScaler(k8sClient),
	}
}

func (s *StopWorkloadsStep) Run(ctx context.Context, dogu *v3beta1.Dogu) stepsv3.StepResult {
	if dogu.Spec.PauseReconciliation || !dogu.Spec.Stopped {
		return stepsv3.Continue()
	}

	// Find the Deployments and StatefulSets belonging to this dogu.
	found, err := s.discovery.Workloads(ctx, dogu.Namespace, dogu.Spec.Name)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
	}

	state, err := s.scaler.ScaleDown(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, found)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
	}
	switch state {
	case workloads.WaitingForSuspension:
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for HelmRelease suspension before stopping workloads")
	case workloads.WaitingForRelease:
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for a suspended HelmRelease for existing workloads")
	case workloads.ScalingDown:
		return s.reportStopProgress(ctx, dogu, true)
	case workloads.ScaledDown:
		return s.reportStopProgress(ctx, dogu, false)
	default:
		// Do not claim any stop progress for a state this step does not know.
		return stepsv3.RequeueWithError(fmt.Errorf("unknown scale state %d", state), v3beta1.ReasonStopping)
	}
}

// reportStopProgress saves the stop status and requests another pass while work remains.
func (s *StopWorkloadsStep) reportStopProgress(ctx context.Context, dogu *v3beta1.Dogu, waiting bool) stepsv3.StepResult {
	status, reason, message := metav1.ConditionTrue, v3beta1.ReasonSucceeded, "All dogu workloads are stopped"
	if waiting {
		status, reason, message = metav1.ConditionFalse, v3beta1.ReasonPartiallyStopped, "Waiting for all dogu workloads and Pods to stop"
	}
	stopped := !waiting
	stoppedChanged := dogu.Status.Stopped != stopped
	dogu.Status.Stopped = stopped

	conditionChanged := meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionStopped,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: dogu.Generation,
	})
	// Save changes to either the Stopped flag or its condition.
	if stoppedChanged || conditionChanged {
		if err := s.k8sClient.Status().Update(ctx, dogu); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonStopping)
		}
	}
	if waiting {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, message)
	}
	return stepsv3.Continue()
}
