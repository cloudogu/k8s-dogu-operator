package install

import (
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCreateConfigStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewCreateConfigStep(newMockDoguConfigRepository(t))

		assert.NotNil(t, step)
	})
}

func TestCreateConfigStep_Run(t *testing.T) {
	tests := []struct {
		name               string
		configRepositoryFn func(t *testing.T) doguConfigRepository
		doguResource       *v2.Dogu
		want               steps.StepResult
	}{
		{
			name: "should fail to get dogu config",
			configRepositoryFn: func(t *testing.T) doguConfigRepository {
				mck := newMockDoguConfigRepository(t)
				mck.EXPECT().Get(testCtx, cescommons.SimpleName("test")).Return(config.DoguConfig{}, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to create dogu config",
			configRepositoryFn: func(t *testing.T) doguConfigRepository {
				mck := newMockDoguConfigRepository(t)
				mck.EXPECT().Get(testCtx, cescommons.SimpleName("test")).Return(config.DoguConfig{}, cloudoguerrors.NewNotFoundError(assert.AnError))
				mck.EXPECT().Create(testCtx, config.CreateDoguConfig("test", make(config.Entries))).Return(config.DoguConfig{}, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to create dogu config",
			configRepositoryFn: func(t *testing.T) doguConfigRepository {
				mck := newMockDoguConfigRepository(t)
				mck.EXPECT().Get(testCtx, cescommons.SimpleName("test")).Return(config.DoguConfig{}, cloudoguerrors.NewNotFoundError(assert.AnError))
				mck.EXPECT().Create(testCtx, config.CreateDoguConfig("test", make(config.Entries))).Return(config.DoguConfig{}, nil)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dcs := &CreateConfigStep{
				configRepository: tt.configRepositoryFn(t),
			}
			assert.Equalf(t, tt.want, dcs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
