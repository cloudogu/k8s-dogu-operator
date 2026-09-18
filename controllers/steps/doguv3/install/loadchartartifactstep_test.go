package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoadChartArtifactStep_Run(t *testing.T) {
	t.Run("loads the repository artifact and marks it available", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		chartLoader := NewMockChartArtifactLoader(t)
		chartLoader.EXPECT().GetChart(testCtx, repository.Status.Artifact.URL, repository.Status.Artifact.Digest).Return(nil, nil)
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).WithStatusSubresource(&v3beta1.Dogu{}).Build()
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, chartLoadedMessage)

		result := NewLoadChartArtifactStep(chartLoader, client, recorder).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Continue(), result)
		require.Len(t, doguResource.Status.Conditions, 1)
		condition := doguResource.Status.Conditions[0]
		assert.Equal(t, v3beta1.ConditionChartAvailable, condition.Type)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
		assert.Equal(t, v3beta1.ReasonSucceeded, condition.Reason)
		persisted := &v3beta1.Dogu{}
		require.NoError(t, client.Get(testCtx, testNamespacedName, persisted))
		require.Len(t, persisted.Status.Conditions, 1)
		assert.Equal(t, metav1.ConditionTrue, persisted.Status.Conditions[0].Status)
	})

	t.Run("requeues a temporary load failure with error, condition and event", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		chartLoader := NewMockChartArtifactLoader(t)
		chartLoader.EXPECT().GetChart(testCtx, repository.Status.Artifact.URL, repository.Status.Artifact.Digest).Return(nil, assert.AnError)
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).WithStatusSubresource(&v3beta1.Dogu{}).Build()
		recorder := NewMockEventRecorder(t)
		expectedMessage := fmt.Sprintf("failed to load OCIRepository artifact: %s", assert.AnError)
		recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, v3beta1.ReasonDownloadFailed, expectedMessage)

		result := NewLoadChartArtifactStep(chartLoader, client, recorder).Run(testCtx, doguResource)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		require.Len(t, doguResource.Status.Conditions, 1)
		condition := doguResource.Status.Conditions[0]
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, v3beta1.ReasonDownloadFailed, condition.Reason)
		assert.Equal(t, expectedMessage, condition.Message)
	})

	t.Run("requeues without error while the artifact URL returns not found", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		notFoundErr := fmt.Errorf("%w: 404 Not Found", errChartArtifactNotFound)
		chartLoader := NewMockChartArtifactLoader(t)
		chartLoader.EXPECT().GetChart(testCtx, repository.Status.Artifact.URL, repository.Status.Artifact.Digest).Return(nil, notFoundErr)
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).WithStatusSubresource(&v3beta1.Dogu{}).Build()

		result := NewLoadChartArtifactStep(chartLoader, client, nil).Run(testCtx, doguResource)

		assert.NoError(t, result.Err)
		assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		require.Len(t, doguResource.Status.Conditions, 1)
		assert.Equal(t, metav1.ConditionFalse, doguResource.Status.Conditions[0].Status)
	})

	t.Run("aborts on an invalid artifact", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		invalidErr := fmt.Errorf("%w: digest mismatch", errInvalidChartArtifact)
		chartLoader := NewMockChartArtifactLoader(t)
		chartLoader.EXPECT().GetChart(testCtx, repository.Status.Artifact.URL, repository.Status.Artifact.Digest).Return(nil, invalidErr)
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).WithStatusSubresource(&v3beta1.Dogu{}).Build()

		result := NewLoadChartArtifactStep(chartLoader, client, nil).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.Abort(v3beta1.ReasonDownloadFailed, "failed to load OCIRepository artifact: invalid chart artifact: digest mismatch"), result)
		require.Len(t, doguResource.Status.Conditions, 1)
		assert.Equal(t, metav1.ConditionFalse, doguResource.Status.Conditions[0].Status)
	})

	t.Run("requeues while the ready repository has no artifact metadata", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		repository.Status.Artifact.Digest = ""
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(repository).Build()

		result := NewLoadChartArtifactStep(nil, client, nil).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "waiting for OCIRepository artifact"), result)
	})

	t.Run("does not load a retained artifact when the repository is no longer ready", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		repository.Status.Conditions[0].Status = metav1.ConditionFalse
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(repository).Build()

		result := NewLoadChartArtifactStep(nil, client, nil).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "waiting for current OCIRepository artifact"), result)
	})

	t.Run("does not load an artifact from an older repository generation", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		repository := readyRepository.DeepCopy()
		repository.Generation++
		client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(repository).Build()

		result := NewLoadChartArtifactStep(nil, client, nil).Run(testCtx, doguResource)

		assert.Equal(t, doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "waiting for current OCIRepository artifact"), result)
	})

	t.Run("requeues while the repository is not visible in the client cache", func(t *testing.T) {
		client := fake.NewClientBuilder().WithScheme(testScheme).Build()

		result := NewLoadChartArtifactStep(nil, client, nil).Run(testCtx, testDoguResource.DeepCopy())

		assert.NoError(t, result.Err)
		assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
	})
}
