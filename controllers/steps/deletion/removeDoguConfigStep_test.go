package deletion

import (
	"fmt"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	registryErrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRemoveDoguConfigStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRemoveDoguConfigStep(newMockDoguConfigRepository(t))

		assert.NotNil(t, step)
	})
}

func TestRemoveDoguConfigStep_Run(t *testing.T) {
	tests := []struct {
		name                   string
		doguConfigRepositoryFn func(t *testing.T) doguConfigRepository
		doguResource           *v2.Dogu
		want                   steps.StepResult
	}{
		{
			name: "should fail to delete dogu config repository",
			doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
				repoMock := newMockDoguConfigRepository(t)
				repoMock.EXPECT().Delete(testCtx, cescommons.SimpleName("test")).Return(assert.AnError)
				return repoMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.StepResult{
				Err: fmt.Errorf("could not delete dogu config: %w", assert.AnError),
			},
		},
		{
			name: "dogu config does not exist anymore",
			doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
				repoMock := newMockDoguConfigRepository(t)
				repoMock.EXPECT().Delete(testCtx, cescommons.SimpleName("test")).Return(registryErrors.NewNotFoundError(assert.AnError))
				return repoMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.StepResult{
				Err:      nil,
				Continue: true,
			},
		},
		{
			name: "should delete dogu config repository",
			doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
				repoMock := newMockDoguConfigRepository(t)
				repoMock.EXPECT().Delete(testCtx, cescommons.SimpleName("test")).Return(nil)
				return repoMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.StepResult{
				Err:      nil,
				Continue: true,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdc := &RemoveDoguConfigStep{
				doguConfigRepository: tt.doguConfigRepositoryFn(t),
			}
			assert.Equalf(t, tt.want, rdc.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
