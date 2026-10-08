package deletion

import (
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	v1 "k8s.io/api/core/v1"
)

// failDeletion emits a deletion-failure warning event and returns a requeue-with-error result, so
// every deletion step reports failures consistently (warning event + ReasonDeletionFailed + requeue).
func failDeletion(recorder EventRecorder, doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	recorder.Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, err.Error())
	return stepsv3.RequeueWithError(err, stepsv3.ReasonDeletionFailed)
}

// escalateDeletionStall emits a warning event and requeues with the DeletionStalled reason. It is used
// when the flux resource being deleted reports a stalled reconciliation (e.g. a failing helm
// uninstall): deletion keeps retrying - because a hard abort would leave the dogu stuck in Terminating
// anyway - but the event and condition surface the root cause so an operator can intervene.
func escalateDeletionStall(recorder EventRecorder, doguResource *v3beta1.Dogu, message string) stepsv3.StepResult {
	recorder.Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionStalled, message)
	return stepsv3.RequeueAfter(defaultRequeueAfter, stepsv3.ReasonDeletionStalled, message)
}
