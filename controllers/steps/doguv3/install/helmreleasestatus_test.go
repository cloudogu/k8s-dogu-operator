package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestNewHelmReleaseStatusStep(t *testing.T) {
	k8sClient := NewMockK8sClient(t)
	recorder := NewMockEventRecorder(t)

	step := NewHelmReleaseStatusStep(k8sClient, recorder)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, recorder, step.eventRecorder)
}

func TestHelmReleaseStatusStep_Run(t *testing.T) {
	helmScheme := runtime.NewScheme()
	_ = v3beta1.AddToScheme(helmScheme)
	_ = fluxhelm.AddToScheme(helmScheme)

	release := &fluxhelm.HelmRelease{
		Namespace: testNamespace,
		Name:      testDoguName,
	}

	t.Run("should requeue if HelmRelease is not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.Contains(t, result.ReadyMessage, "not found")
		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
	})

	t.Run("should requeue with error if HelmRelease cannot be fetched", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return assert.AnError
			},
		}).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		require.Error(t, result.Err)
		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to get HelmRelease")
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.False(t, result.Continue)
	})

	t.Run("should continue if HelmRelease exists", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(release.DeepCopy()).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		assert.Equal(t, stepsv3.Continue(), result)
	})
}
