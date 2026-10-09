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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func helmStatusScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v3beta1.AddToScheme(scheme)
	_ = fluxhelm.AddToScheme(scheme)
	return scheme
}

func TestNewHelmReleaseStatusStep(t *testing.T) {
	k8sClient := NewMockK8sClient(t)
	recorder := NewMockEventRecorder(t)

	step := NewHelmReleaseStatusStep(k8sClient, recorder)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, recorder, step.eventRecorder)
}

func TestHelmReleaseStatusStep_Run(t *testing.T) {
	helmScheme := helmStatusScheme()

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

	t.Run("should continue if the desired chart version is deployed and the dogu is healthy", func(t *testing.T) {
		deployed := release.DeepCopy()
		deployed.Generation = 1
		deployed.Status.ObservedGeneration = 1
		deployed.Status.History = fluxhelm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: testVersion + "+ebb48ffcdce3"}}
		doguResource := testDoguResource.DeepCopy()
		doguResource.Status.Conditions = []metav1.Condition{healthyCondition(metav1.ConditionTrue, "all workloads are ready")}
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(deployed).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t))

		result := step.Run(context.Background(), doguResource)

		assert.Equal(t, stepsv3.Continue(), result)
	})
}

func healthyCondition(status metav1.ConditionStatus, message string) metav1.Condition {
	return metav1.Condition{Type: v3beta1.ConditionHealthy, Status: status, Reason: "SomeReason", Message: message}
}

func TestHelmReleaseStatusStep_stepResultForRelease(t *testing.T) {
	tests := []struct {
		name  string
		state fluxstate.ReleaseState
		want  stepsv3.StepResult
	}{
		{name: "chart unavailable", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseChartUnavailable, Message: "msg"}, want: stepsv3.Abort(v3beta1.ReasonDownloadFailed, "msg")},
		{name: "installing", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseInstalling, Message: "msg"}, want: stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "msg")},
		{name: "upgrading", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseUpgrading, Message: "msg", EverDeployed: true}, want: stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonUpgrading, "msg")},
		{name: "install failed", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseInstallFailed, Message: "msg"}, want: stepsv3.Abort(v3beta1.ReasonInstallFailed, "msg")},
		{name: "upgrade failed", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseUpgradeFailed, Message: "msg", EverDeployed: true}, want: stepsv3.Abort(v3beta1.ReasonUpgradeFailed, "msg")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t))

			got := step.stepResultForRelease(tt.state, testDoguResource.DeepCopy())

			assert.Equal(t, tt.want, got)
		})
	}

	deployedTests := []struct {
		name         string
		everDeployed bool
		stopped      bool
		conditions   []metav1.Condition
		want         stepsv3.StepResult
	}{
		{
			name:       "deployed and healthy",
			conditions: []metav1.Condition{healthyCondition(metav1.ConditionTrue, "all workloads are ready")},
			want:       stepsv3.Continue(),
		},
		{
			name:       "deployed but not healthy",
			conditions: []metav1.Condition{healthyCondition(metav1.ConditionFalse, "deployment ldap is not ready")},
			want:       stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "deployment ldap is not ready"),
		},
		{
			name:         "upgrade deployed but not healthy",
			everDeployed: true,
			conditions:   []metav1.Condition{healthyCondition(metav1.ConditionFalse, "deployment ldap is not ready")},
			want:         stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "deployment ldap is not ready"),
		},
		{
			name:       "deployed but health unknown",
			conditions: []metav1.Condition{healthyCondition(metav1.ConditionUnknown, "operator shut down")},
			want:       stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "operator shut down"),
		},
		{
			name: "deployed but Healthy condition missing",
			want: stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "Waiting for the health check of the dogu"),
		},
		{
			name:       "deployed and stopped is ready",
			stopped:    true,
			conditions: []metav1.Condition{healthyCondition(metav1.ConditionTrue, "all workloads are ready")},
			want:       stepsv3.Continue(),
		},
	}

	for _, tt := range deployedTests {
		t.Run(tt.name, func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			doguResource.Spec.Stopped = tt.stopped
			doguResource.Status.Conditions = tt.conditions
			// the mocks fail on any unexpected call, e.g. a status update
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t))
			state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: tt.everDeployed}

			got := step.stepResultForRelease(state, doguResource)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("unknown phase", func(t *testing.T) {
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t))

		got := step.stepResultForRelease(fluxstate.ReleaseState{Phase: "Unknown"}, testDoguResource.DeepCopy())

		assert.ErrorContains(t, got.Err, `unknown release phase "Unknown"`)
		assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
		assert.False(t, got.Continue)
	})
}
