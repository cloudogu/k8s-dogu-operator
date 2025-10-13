package install

import (
	"testing"
	"time"

	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewFinalizerExistsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewCreateFinalizerStep(newMockK8sClient(t))

		assert.NotNil(t, step)
	})
}

func TestFinalizerExistsStep_Run(t *testing.T) {
	t.Run("Successfully added finalizer", func(t *testing.T) {
		doguResource := &v2.Dogu{}
		client := newMockK8sClient(t)
		client.EXPECT().Get(testCtx, client2.ObjectKeyFromObject(doguResource), doguResource).Return(nil)
		client.EXPECT().Update(testCtx, doguResource).Return(nil)
		sut := NewCreateFinalizerStep(client)

		result := sut.Run(testCtx, doguResource)

		assert.NotNil(t, sut)
		assert.Equal(t, true, result.Continue)
		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, nil, result.Err)
	})
	t.Run("fail to add finalizer", func(t *testing.T) {
		doguResource := &v2.Dogu{}
		client := newMockK8sClient(t)
		client.EXPECT().Get(testCtx, client2.ObjectKeyFromObject(doguResource), doguResource).Return(nil)
		client.EXPECT().Update(testCtx, doguResource).Return(assert.AnError)
		sut := NewCreateFinalizerStep(client)

		result := sut.Run(testCtx, doguResource)

		assert.ErrorIs(t, result.Err, assert.AnError)
	})
}
