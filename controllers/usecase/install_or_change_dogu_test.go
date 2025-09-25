package usecase

import (
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewDoguInstallOrChangeUseCase(t *testing.T) {
	t.Run("Successfully created install or change usecase with correct order", func(t *testing.T) {
		usecase := NewDoguInstallOrChangeUseCase(
			NewMockK8sClient(t),
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
		clientFn         func(t *testing.T) client.Client
		stepsFn          func(t *testing.T) []Step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get dogu resource",
			clientFn: func(t *testing.T) client.Client {
				scheme := runtime.NewScheme()
				err := v2.AddToScheme(scheme)
				require.NoError(t, err)
				mck := fake.NewClientBuilder().
					WithScheme(scheme).
					Build()

				return mck
			},
			stepsFn: func(t *testing.T) []Step {
				return []Step{NewMockStep(t)}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.Error,
		},
		{
			name: "should requeue run on requeueAfter time",
			clientFn: func(t *testing.T) client.Client {
				scheme := runtime.NewScheme()
				err := v2.AddToScheme(scheme)
				require.NoError(t, err)
				dogu := &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}
				mck := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(dogu).
					Build()

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
			clientFn: func(t *testing.T) client.Client {
				scheme := runtime.NewScheme()
				err := v2.AddToScheme(scheme)
				require.NoError(t, err)
				dogu := &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}
				mck := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(dogu).
					Build()

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
			clientFn: func(t *testing.T) client.Client {
				scheme := runtime.NewScheme()
				err := v2.AddToScheme(scheme)
				require.NoError(t, err)
				dogu := &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}
				mck := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(dogu).
					Build()

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
			clientFn: func(t *testing.T) client.Client {
				scheme := runtime.NewScheme()
				err := v2.AddToScheme(scheme)
				require.NoError(t, err)
				dogu := &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}
				mck := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(dogu).
					Build()

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
			dicu := &DoguInstallOrChangeUseCase{
				client: tt.clientFn(t),
				steps:  tt.stepsFn(t),
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
