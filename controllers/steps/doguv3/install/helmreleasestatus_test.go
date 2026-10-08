package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
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

	t.Run("should requeue while the release is installing", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(release.DeepCopy()).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
	})

	t.Run("should continue if the desired chart version is deployed", func(t *testing.T) {
		deployed := release.DeepCopy()
		deployed.Generation = 1
		deployed.Status.ObservedGeneration = 1
		deployed.Status.History = fluxhelm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: testVersion + "+ebb48ffcdce3"}}
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(deployed).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		assert.Equal(t, stepsv3.Continue(), result)
	})
}

func TestStepResultForRelease(t *testing.T) {
	tests := []struct {
		name  string
		phase fluxstate.ReleasePhase
		want  stepsv3.StepResult
	}{
		{name: "chart unavailable", phase: fluxstate.PhaseChartUnavailable, want: stepsv3.Abort(v3beta1.ReasonDownloadFailed, "msg")},
		{name: "installing", phase: fluxstate.PhaseInstalling, want: stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "msg")},
		{name: "upgrading", phase: fluxstate.PhaseUpgrading, want: stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonUpgrading, "msg")},
		{name: "install failed", phase: fluxstate.PhaseInstallFailed, want: stepsv3.Abort(ReasonInstallFailed, "msg")},
		{name: "upgrade failed", phase: fluxstate.PhaseUpgradeFailed, want: stepsv3.Abort(ReasonUpgradeFailed, "msg")},
		{name: "deployed", phase: fluxstate.PhaseDeployed, want: stepsv3.Continue()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepResultForRelease(fluxstate.ReleaseState{Phase: tt.phase, Message: "msg"})

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("unknown phase", func(t *testing.T) {
		got := stepResultForRelease(fluxstate.ReleaseState{Phase: "Unknown"})

		assert.ErrorContains(t, got.Err, `unknown release phase "Unknown"`)
		assert.False(t, got.Continue)
	})
}
