package deletion

import (
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
)

// defaultRequeueAfter is the polling interval used while waiting for flux resources to be fully deleted.
const defaultRequeueAfter = 5 * time.Second

// waitForDeletion requeues until the named resource has fully disappeared, so deletion steps poll
// for completion consistently (same interval, same ReasonDeleting, same message format).
func waitForDeletion(kind, name string) stepsv3.StepResult {
	return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonDeleting, fmt.Sprintf("waiting for %s %q to be deleted", kind, name))
}
