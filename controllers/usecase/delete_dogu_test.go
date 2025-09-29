package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func TestNewDoguDeleteUsecase(t *testing.T) {
	t.Run("Successfully created delete usecase with correct order", func(t *testing.T) {
		usecase := NewDoguDeleteUseCase(
			nil,
			nil,
			nil,
			nil,
		)

		assert.NotNil(t, usecase)
	})
}

func TestDoguDeleteUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		stepsFn          func(t *testing.T) []Step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should requeue run on requeueAfter time",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueAfter(2))
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 2,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
		{
			name: "should requeue run on error",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueWithError(assert.AnError))
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.Error,
		},
		{
			name: "should continue after step",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Continue())
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     true,
			wantErr:          assert.NoError,
		},
		{
			name: "should abort after step",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Abort())
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ddu := &DoguDeleteUseCase{
				steps: tt.stepsFn(t),
			}
			got, got1, err := ddu.HandleUntilApplied(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.wantRequeueAfter, got, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
			assert.Equalf(t, tt.wantContinue, got1, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
