package deletion

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testTime = metav1.Time{Time: time.Unix(112313, 0)}

func TestNewStatusStep(t *testing.T) {
	step := NewStatusStep(newMockDoguInterface(t))
	assert.NotEmpty(t, step)
}

func TestStatusStep_Run(t *testing.T) {
	expectedStatus := v2.DoguStatus{
		Status: "deleting",
		Health: "unavailable",
		Conditions: []metav1.Condition{
			{
				Type:               "healthy",
				Status:             metav1.ConditionFalse,
				Reason:             "Deleting",
				Message:            "The dogu is being deleted.",
				LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
			},
			{
				Type:               "ready",
				Status:             metav1.ConditionFalse,
				Reason:             "Deleting",
				Message:            "The dogu is being deleted.",
				LastTransitionTime: metav1.Time{Time: time.Unix(112313, 0)},
			},
		},
	}
	expectedDogu := &v2.Dogu{Status: expectedStatus}
	tests := []struct {
		name            string
		doguInterfaceFn func(t *testing.T) doguInterface
		resource        *v2.Dogu
		want            steps.StepResult
	}{
		{
			name: "should fail to update status",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, &v2.Dogu{}, mock.Anything, metav1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, expectedStatus, status)
				}).Return(nil, assert.AnError)
				return mck
			},
			resource: &v2.Dogu{},
			want:     steps.StepResult{Err: fmt.Errorf("failed to update status of dogu when deleting: %w", assert.AnError)},
		},
		{
			name: "should succeed to update status",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				mck.EXPECT().UpdateStatusWithRetry(testCtx, &v2.Dogu{}, mock.Anything, metav1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, expectedStatus, status)
				}).Return(expectedDogu, nil)
				return mck
			},
			resource: &v2.Dogu{},
			want:     steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldNow := steps.Now
			defer func() { steps.Now = oldNow }()
			steps.Now = func() metav1.Time {
				return testTime
			}

			s := &StatusStep{
				doguInterface: tt.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, s.Run(testCtx, tt.resource), "Run(%v, %v)", testCtx, tt.resource)
		})
	}
}
