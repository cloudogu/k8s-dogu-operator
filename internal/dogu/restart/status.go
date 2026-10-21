package restart

import (
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsSucceeded returns true if the restart of a v3 dogu is finished successfully.
func IsSucceeded(restart *doguv2.DoguRestart) bool {
	return meta.IsStatusConditionTrue(restart.Status.Conditions, doguv2.DoguRestartConditionReady)
}

// IsFailed returns true if the restart of a v3 dogu failed permanently and will never be retried.
func IsFailed(restart *doguv2.DoguRestart) bool {
	ready := meta.FindStatusCondition(restart.Status.Conditions, doguv2.DoguRestartConditionReady)
	if ready == nil || ready.Status != metav1.ConditionFalse {
		return false
	}

	return ready.Reason == doguv2.DoguRestartReasonDoguNotFound
}

// IsTerminal returns true if the restart of a v3 dogu is finished, successfully or not.
// A terminal restart never changes the workloads of the dogu again.
func IsTerminal(restart *doguv2.DoguRestart) bool {
	return IsSucceeded(restart) || IsFailed(restart)
}
