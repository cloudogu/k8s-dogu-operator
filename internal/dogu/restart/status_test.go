package restart

import (
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func restartWithReady(status metav1.ConditionStatus, reason string) *doguv2.DoguRestart {
	return &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{Conditions: []metav1.Condition{
		{Type: doguv2.DoguRestartConditionReady, Status: status, Reason: reason},
	}}}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name          string
		restart       *doguv2.DoguRestart
		wantSucceeded bool
		wantFailed    bool
	}{
		{name: "new restart", restart: &doguv2.DoguRestart{}},
		{name: "v2 restart", restart: &doguv2.DoguRestart{Status: doguv2.DoguRestartStatus{Phase: doguv2.RestartStatusPhaseCompleted}}},
		{name: "rollout in progress", restart: restartWithReady(metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress)},
		{name: "dogu not idle", restart: restartWithReady(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle)},
		{name: "patch failed", restart: restartWithReady(metav1.ConditionFalse, doguv2.DoguRestartReasonPatchFailed)},
		{name: "succeeded", restart: restartWithReady(metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded), wantSucceeded: true},
		{name: "superseded", restart: restartWithReady(metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded), wantSucceeded: true},
		{name: "dogu stopped", restart: restartWithReady(metav1.ConditionTrue, doguv2.DoguRestartReasonDoguStopped), wantSucceeded: true},
		{name: "dogu not found", restart: restartWithReady(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound), wantFailed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantSucceeded, IsSucceeded(tt.restart))
			assert.Equal(t, tt.wantFailed, IsFailed(tt.restart))
			assert.Equal(t, tt.wantSucceeded || tt.wantFailed, IsTerminal(tt.restart))
		})
	}
}
