package deletion

import (
	"context"
	"reflect"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testCtx = context.Background()

const namespace = "ecosystem"

func TestNewDeleteOutOfHealthConfigMapStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeleteOutOfHealthConfigMapStep(newMockK8sClient(t))

		assert.NotNil(t, step)
	})
}

func TestDeleteOutOfHealthConfigMapStep_Run(t *testing.T) {
	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to update configmap",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				cmKey := types.NamespacedName{Namespace: namespace, Name: configMapName}
				clientMock.EXPECT().Get(testCtx, cmKey, mock.Anything, &client.GetOptions{}).Return(assert.AnError)
				clientMock.EXPECT().Update(testCtx, mock.Anything, &client.UpdateOptions{}).Return(assert.AnError)
				return clientMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.StepResult{Err: assert.AnError},
		},
		{
			name: "successfully update configmap",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				cmKey := types.NamespacedName{Namespace: namespace, Name: configMapName}
				clientMock.EXPECT().Get(testCtx, cmKey, mock.Anything, &client.GetOptions{}).Return(assert.AnError)
				clientMock.EXPECT().Update(testCtx, mock.Anything, &client.UpdateOptions{}).Return(nil)
				return clientMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      "test",
				},
			},
			want: steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dhc := &DeleteOutOfHealthConfigMapStep{
				client: tt.clientFn(t),
			}
			if got := dhc.Run(testCtx, tt.doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}
		})
	}
}
