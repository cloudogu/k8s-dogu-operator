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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testCtx = context.Background()

func TestNewDoguDeleteUsecase(t *testing.T) {
	t.Run("Successfully created delete usecase with correct order", func(t *testing.T) {
		usecase := NewDoguDeleteUseCase(
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

func TestDoguDeleteUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		clientFn         func(t *testing.T) K8sClient
		stepsFn          func(t *testing.T) []Step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get resource",
			clientFn: func(t *testing.T) K8sClient {
				mck := NewMockK8sClient(t)
				mck.EXPECT().Get(testCtx, client.ObjectKey{Name: "test"}, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(assert.AnError)
				return mck
			},
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "should requeue run on requeueAfter time",
			clientFn: func(t *testing.T) K8sClient {
				mck := NewMockK8sClient(t)
				mck.EXPECT().Get(testCtx, client.ObjectKey{Name: "test"}, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(nil)
				return mck
			},
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
			clientFn: func(t *testing.T) K8sClient {
				mck := NewMockK8sClient(t)
				mck.EXPECT().Get(testCtx, client.ObjectKey{Name: "test"}, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(nil)
				return mck
			},
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
			clientFn: func(t *testing.T) K8sClient {
				mck := NewMockK8sClient(t)
				mck.EXPECT().Get(testCtx, client.ObjectKey{Name: "test"}, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(nil)
				return mck
			},
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
			clientFn: func(t *testing.T) K8sClient {
				mck := NewMockK8sClient(t)
				mck.EXPECT().Get(testCtx, client.ObjectKey{Name: "test"}, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(nil)
				return mck
			},
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
				client: tt.clientFn(t),
				steps:  tt.stepsFn(t),
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
