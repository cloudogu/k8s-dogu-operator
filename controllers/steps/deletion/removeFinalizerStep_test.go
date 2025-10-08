package deletion

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRemoveFinalizerStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRemoveFinalizerStep(newMockK8sClient(t))

		assert.NotNil(t, step)
	})
}

func TestRemoveFinalizerStep_Run(t *testing.T) {
	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to update dogu resource",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				clientMock.EXPECT().Update(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{
						Finalizers: []string{},
					},
				}).Return(assert.AnError)
				return clientMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Finalizers: []string{legacyFinalizerName},
				},
			},
			want: steps.StepResult{
				Err: fmt.Errorf("failed to update dogu: %w", assert.AnError),
			},
		},
		{
			name: "should remove finalizer from dogu resource",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				clientMock.EXPECT().Update(testCtx, &v2.Dogu{
					ObjectMeta: v1.ObjectMeta{
						Finalizers: []string{},
					},
				}).Return(nil)
				return clientMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Finalizers: []string{
						legacyFinalizerName,
						finalizerName,
					},
				},
			},
			want: steps.StepResult{
				Continue: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := &RemoveFinalizerStep{
				client: tt.clientFn(t),
			}
			assert.Equalf(t, tt.want, rf.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
