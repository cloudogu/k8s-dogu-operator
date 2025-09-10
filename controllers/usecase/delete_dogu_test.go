package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func TestNewDoguDeleteUsecase(t *testing.T) {
	t.Run("Successfully created delete usacase", func(t *testing.T) {
		usecase := NewDoguDeleteUsecase(newMockK8sClient(t), &util.ManagerSet{
			DoguRegistrator: &cesregistry.CesDoguRegistrator{},
		}, util.ConfigRepositories{
			DoguConfigRepository:    &repository.DoguConfigRepository{},
			SensitiveDoguRepository: &repository.DoguConfigRepository{},
		}, &config.OperatorConfig{})

		assert.NotNil(t, usecase)
	})
}

func TestDoguDeleteUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		stepsFn          func(t *testing.T) []step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should return requeue after time duration",
			stepsFn: func(t *testing.T) []step {
				firstStep := newMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.RequeueAfter(time.Second * 3))
				return []step{firstStep}
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
			stepsFn: func(t *testing.T) []step {
				firstStep := newMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.RequeueWithError(assert.AnError))
				return []step{firstStep}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: time.Duration(0),
			wantContinue:     true,
			wantErr:          assert.Error,
		},
		{
			name: "should abort step loop",
			stepsFn: func(t *testing.T) []step {
				firstStep := newMockStep(t)
				firstStep.EXPECT().Run(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{Name: "test"},
				}).Return(steps.Abort())
				return []step{firstStep}
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
			stepsFn: func(t *testing.T) []step {
				stepLoop := []step{}
				for i := 0; i < 10; i++ {
					s := newMockStep(t)
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
