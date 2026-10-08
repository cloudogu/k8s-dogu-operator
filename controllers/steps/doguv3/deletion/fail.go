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
