package install

import (
	"context"
	"testing"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespace = "ecosystem"

var testCtx = context.Background()

func TestNewConditionsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewConditionsStep(nil)

		assert.NotNil(t, step)
	})
}

func TestConditionsStep_Run(t *testing.T) {
	tests := []struct {
		name               string
		doguInterfaceFn    func(t *testing.T) doguInterface
		conditionUpdaterFn func(t *testing.T) ConditionUpdater
		doguResource       *doguv2.Dogu
		want               steps.StepResult
	}{
		{
			name: "no conditions are set",
			conditionUpdaterFn: func(t *testing.T) ConditionUpdater {
				updaterMock := NewMockConditionUpdater(t)
				updaterMock.EXPECT().UpdateConditions(testCtx, mock.Anything, mock.Anything).Return(nil)
				return updaterMock
			},
			doguResource: &doguv2.Dogu{
				Status: doguv2.DoguStatus{},
			},
			want: steps.Continue(),
		},
		{
			name: "all conditions are set",
			conditionUpdaterFn: func(t *testing.T) ConditionUpdater {
				return NewMockConditionUpdater(t)
			},
			doguResource: &doguv2.Dogu{
				Status: doguv2.DoguStatus{
					Conditions: []v1.Condition{
						{
							Type:    doguv2.ConditionHealthy,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionMeetsMinVolumeSize,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionReady,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionSupportMode,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionPauseReconciliation,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
					},
				},
			},
			want: steps.Continue(),
		},
		{
			name: "one condition is not set",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				return newMockDoguInterface(t)
			},
			conditionUpdaterFn: func(t *testing.T) ConditionUpdater {
				updaterMock := NewMockConditionUpdater(t)
				updaterMock.EXPECT().UpdateConditions(testCtx, mock.Anything, mock.Anything).Return(nil)
				return updaterMock
			},
			doguResource: &doguv2.Dogu{
				Status: doguv2.DoguStatus{
					Conditions: []v1.Condition{
						{
							Type:    doguv2.ConditionHealthy,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionMeetsMinVolumeSize,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionSupportMode,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
					},
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to set condition",
			doguInterfaceFn: func(t *testing.T) doguInterface {
				return newMockDoguInterface(t)
			},
			conditionUpdaterFn: func(t *testing.T) ConditionUpdater {
				updaterMock := NewMockConditionUpdater(t)
				updaterMock.EXPECT().UpdateConditions(testCtx, mock.Anything, mock.Anything).Return(assert.AnError)
				return updaterMock
			},
			doguResource: &doguv2.Dogu{
				Status: doguv2.DoguStatus{
					Conditions: []v1.Condition{
						{
							Type:    doguv2.ConditionHealthy,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionMeetsMinVolumeSize,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
						{
							Type:    doguv2.ConditionSupportMode,
							Status:  v1.ConditionTrue,
							Reason:  "TestReason",
							Message: "TestMessage",
						},
					},
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ConditionsStep{
				conditionUpdater: tt.conditionUpdaterFn(t),
			}
			assert.Equalf(t, tt.want, cs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
