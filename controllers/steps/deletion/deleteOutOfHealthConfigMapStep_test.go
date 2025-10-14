package deletion

import (
	"context"
	"reflect"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

const namespace = "ecosystem"

func TestNewDeleteOutOfHealthConfigMapStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeleteOutOfHealthConfigMapStep(newMockDoguHealthStatusUpdater(t))

		assert.NotNil(t, step)
	})
}

func TestDeleteOutOfHealthConfigMapStep_Run(t *testing.T) {
	tests := []struct {
		name                      string
		doguHealthStatusUpdaterFn func(t *testing.T) doguHealthStatusUpdater
		doguResource              *v2.Dogu
		want                      steps.StepResult
	}{
		{
			name: "should fail to update configmap",
			doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
				mck := newMockDoguHealthStatusUpdater(t)
				mck.EXPECT().DeleteDoguOutOfHealthConfigMap(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "test",
					},
				}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "successfully update configmap",
			doguHealthStatusUpdaterFn: func(t *testing.T) doguHealthStatusUpdater {
				mck := newMockDoguHealthStatusUpdater(t)
				mck.EXPECT().DeleteDoguOutOfHealthConfigMap(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{
						Namespace: namespace,
						Name:      "test",
					},
				}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dhc := &DeleteOutOfHealthConfigMapStep{
				doguHealthStatusUpdater: tt.doguHealthStatusUpdaterFn(t),
			}
			if got := dhc.Run(testCtx, tt.doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}
