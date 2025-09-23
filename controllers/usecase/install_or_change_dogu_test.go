package usecase

import (
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguInstallOrChangeUseCase(t *testing.T) {
	t.Run("Successfully created install or change usecase with correct order", func(t *testing.T) {
		usecase := NewDoguInstallOrChangeUseCase(
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

		assert.NotNil(t, usecase)
	})
}

func TestDoguInstallOrChangeUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		stepsFn          func(t *testing.T) []Step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should return requeue after time duration",
			stepsFn: func(t *testing.T) []Step {
				firstStep := NewMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.RequeueAfter(time.Second * 3))
				return []Step{firstStep}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: time.Second * 3,
			wantContinue:     true,
			wantErr:          assert.NoError,
		},
		{
			name: "should return error",
			stepsFn: func(t *testing.T) []Step {
				firstStep := NewMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.RequeueWithError(assert.AnError))
				return []Step{firstStep}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: time.Duration(0),
			wantContinue:     true,
			wantErr:          assert.Error,
		},
		{
			name: "should abort Step loop",
			stepsFn: func(t *testing.T) []Step {
				firstStep := NewMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.Abort())
				return []Step{firstStep}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: time.Duration(0),
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
		{
			name: "should loop through all steps",
			stepsFn: func(t *testing.T) []Step {
				stepLoop := []Step{}
				for i := 0; i < 10; i++ {
					s := NewMockStep(t)
					s.EXPECT().Run(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(steps.Continue())
					stepLoop = append(stepLoop, s)
				}

				return stepLoop
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: time.Duration(0),
			wantContinue:     true,
			wantErr:          assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dicu := &DoguInstallOrChangeUseCase{
				steps: tt.stepsFn(t),
			}
			got, got1, err := dicu.HandleUntilApplied(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.wantRequeueAfter, got, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
			assert.Equalf(t, tt.wantContinue, got1, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
