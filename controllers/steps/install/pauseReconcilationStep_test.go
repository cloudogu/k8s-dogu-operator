package install

import (
	"context"
	"github.com/stretchr/testify/mock"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewPauseReconcilationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewPauseReconciliationStep(newMockDoguInterface(t))

		assert.NotNil(t, step)
	})
}

func TestPauseReconciliationStep_Run(t *testing.T) {
	tests := []struct {
		name            string
		doguResource    *v2.Dogu
		doguInterfaceFn func(t *testing.T) doguInterface
		want            steps.StepResult
	}{
		{
			name: "should fail to set condition",
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					PauseReconciliation: true,
				},
			},
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				condition := v1.Condition{
					Type:               v2.ConditionPauseReconciliation,
					Status:             v1.ConditionTrue,
					Reason:             conditionReasonPaused,
					Message:            conditionMessagePaused,
					LastTransitionTime: v1.Now().Rfc3339Copy(),
				}
				dogu := &v2.Dogu{
					Spec: v2.DoguSpec{
						PauseReconciliation: true,
					},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{condition},
					},
				}
				mck.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, v1.UpdateOptions{}).Return(nil, assert.AnError)
				return mck
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should abort because of active pause reconcilation",
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					PauseReconciliation: true,
				},
			},
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				condition := v1.Condition{
					Type:               v2.ConditionPauseReconciliation,
					Status:             v1.ConditionTrue,
					Reason:             conditionReasonPaused,
					Message:            conditionMessagePaused,
					LastTransitionTime: v1.Now().Rfc3339Copy(),
				}
				dogu := &v2.Dogu{
					Spec: v2.DoguSpec{
						PauseReconciliation: true,
					},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{condition},
					},
				}
				mck.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) {
					modifyStatusFn(dogu.Status)
					assert.Equal(t, v1.Condition{
						Type:               v2.ConditionPauseReconciliation,
						Status:             v1.ConditionTrue,
						Reason:             conditionReasonPaused,
						Message:            conditionMessagePaused,
						LastTransitionTime: v1.Now().Rfc3339Copy(),
					}, dogu.Status.Conditions[0])
				}).Return(dogu, nil)
				return mck
			},
			want: steps.Abort(),
		},
		{
			name: "should continue because of inactive pause reconcilation",
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					PauseReconciliation: false,
				},
			},
			doguInterfaceFn: func(t *testing.T) doguInterface {
				mck := newMockDoguInterface(t)
				condition := v1.Condition{
					Type:               v2.ConditionPauseReconciliation,
					Status:             v1.ConditionFalse,
					Reason:             conditionReasonNotPaused,
					Message:            conditionMessageNotPaused,
					LastTransitionTime: v1.Now().Rfc3339Copy(),
				}
				dogu := &v2.Dogu{
					Spec: v2.DoguSpec{
						PauseReconciliation: false,
					},
					Status: v2.DoguStatus{
						Conditions: []v1.Condition{condition},
					},
				}
				mck.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, v1.UpdateOptions{}).Return(dogu, nil)
				return mck
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs := &PauseReconciliationStep{
				doguInterface: tt.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, prs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
