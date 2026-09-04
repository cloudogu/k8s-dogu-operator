package v3

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func TestDoguUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		stepsFn          func(t *testing.T) []steps.Step
		doguResource     *v3beta1.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should requeue run on requeueAfter time",
			stepsFn: func(t *testing.T) []steps.Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueAfter(2))
				return []steps.Step{step}
			},
			doguResource: &v3beta1.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 2,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
		{
			name: "should requeue run on error",
			stepsFn: func(t *testing.T) []steps.Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueWithError(assert.AnError))
				return []steps.Step{step}
			},
			doguResource: &v3beta1.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.Error,
		},
		{
			name: "should continue after step",
			stepsFn: func(t *testing.T) []steps.Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Continue())
				return []steps.Step{step}
			},
			doguResource: &v3beta1.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     true,
			wantErr:          assert.NoError,
		},
		{
			name: "should abort after step",
			stepsFn: func(t *testing.T) []steps.Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Abort())
				return []steps.Step{step}
			},
			doguResource: &v3beta1.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duc := &DoguUseCase{
				steps: tt.stepsFn(t),
			}
			got, got1, err := duc.HandleUntilApplied(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.wantRequeueAfter, got, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
			assert.Equalf(t, tt.wantContinue, got1, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func TestNewDoguInstallOrChangeUseCase(t *testing.T) {
	dummyStep := &install.DummyStep{}

	got := NewDoguInstallOrChangeUseCase(
		dummyStep,
	)

	wantTypes := []string{
		"*install.DummyStep",
	}

	assert.NotNil(t, got)
	require.True(t,
		slices.Equal(typesOf(got.steps), wantTypes),
		"order mismatch: got=%v want=%v",
		typesOf(got.steps), wantTypes,
	)
}

func TestNewDoguDeleteUseCase(t *testing.T) {
	got := NewDoguDeleteUseCase()

	wantTypes := []string{}

	assert.NotNil(t, got)
	require.True(t,
		slices.Equal(typesOf(got.steps), wantTypes),
		"order mismatch: got=%v want=%v",
		typesOf(got.steps), wantTypes,
	)
}

func typesOf[T any](xs []T) []string {
	out := make([]string, len(xs))
	for i, v := range xs {
		out[i] = reflect.TypeOf(v).String()
	}
	return out
}
