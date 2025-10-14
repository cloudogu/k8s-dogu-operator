package deletion

import (
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	doguWithLegacyFinalizer := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Namespace:  "test",
			Name:       "dogu",
			Finalizers: []string{legacyFinalizerName},
		},
	}

	doguWithFinalizer := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Namespace:  "test",
			Name:       "dogu",
			Finalizers: []string{finalizerName},
		},
	}

	doguWithoutFinalizer := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Namespace:  "test",
			Name:       "dogu",
			Finalizers: []string{},
		},
	}

	tests := []struct {
		name         string
		clientFn     func(t *testing.T) k8sClient
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should skip if dogu has no finalizer",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				return clientMock
			},
			doguResource: doguWithoutFinalizer,
			want:         steps.Continue(),
		},
		{
			name: "should fail to update dogu resource",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				clientMock.EXPECT().Get(testCtx, client.ObjectKeyFromObject(doguWithoutFinalizer), doguWithLegacyFinalizer).Return(nil)
				clientMock.EXPECT().Update(testCtx, doguWithoutFinalizer).Return(assert.AnError)
				return clientMock
			},
			doguResource: doguWithLegacyFinalizer,
			want: steps.StepResult{
				Err: fmt.Errorf("failed to update dogu: %w", assert.AnError),
			},
		},
		{
			name: "should remove finalizer",
			clientFn: func(t *testing.T) k8sClient {
				clientMock := newMockK8sClient(t)
				clientMock.EXPECT().Get(testCtx, client.ObjectKeyFromObject(doguWithFinalizer), doguWithFinalizer).Return(nil)
				clientMock.EXPECT().Update(testCtx, doguWithoutFinalizer).Return(nil)
				return clientMock
			},
			doguResource: doguWithFinalizer,
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
