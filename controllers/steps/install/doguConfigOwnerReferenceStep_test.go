package install

import (
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguConfigOwnerReferenceStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDoguConfigOwnerReferenceStep(
			util.ConfigRepositories{
				DoguConfigRepository: &repository.DoguConfigRepository{},
			},
		)

		assert.NotNil(t, step)
	})
}

func TestDoguConfigOwnerReferenceStep_Run(t *testing.T) {
	tests := []struct {
		name                   string
		doguConfigRepositoryFn func(t *testing.T) doguConfigRepository
		doguResource           *v2.Dogu
		want                   steps.StepResult
	}{
		{
			name: "should fail to set owner reference",
			doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
				mck := newMockDoguConfigRepository(t)
				mck.EXPECT().SetOwnerReference(testCtx, cescommons.SimpleName("test"), []metav1.OwnerReference{
					{
						Name:       "test",
						Kind:       "Dogu",
						APIVersion: "v1",
						UID:        "uid",
						Controller: &[]bool{true}[0],
					},
				}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "uid",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Dogu",
					APIVersion: "v1",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to set owner reference",
			doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
				mck := newMockDoguConfigRepository(t)
				mck.EXPECT().SetOwnerReference(testCtx, cescommons.SimpleName("test"), []metav1.OwnerReference{
					{
						Name:       "test",
						Kind:       "Dogu",
						APIVersion: "v1",
						UID:        "uid",
						Controller: &[]bool{true}[0],
					},
				}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
					UID:  "uid",
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "Dogu",
					APIVersion: "v1",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dcs := &DoguConfigOwnerReferenceStep{
				doguConfigRepository: tt.doguConfigRepositoryFn(t),
			}
			assert.Equalf(t, tt.want, dcs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
