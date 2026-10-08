package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/health"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type stubHealthChecker struct {
	state health.State
	err   error
}

func (s stubHealthChecker) Check(_ context.Context, _ *v3beta1.Dogu) (health.State, error) {
	return s.state, s.err
}

var (
	healthyState   = health.State{Healthy: true, Reason: v3beta1.ReasonSucceeded, Message: "all workloads are ready"}
	unhealthyState = health.State{Healthy: false, Reason: v3beta1.ReasonWorkloadsNotReady, Message: "workloads not ready"}
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
	checker := stubHealthChecker{}

	step := NewHelmReleaseStatusStep(k8sClient, recorder, checker)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, recorder, step.eventRecorder)
	assert.Equal(t, checker, step.healthChecker)
}

func TestHelmReleaseStatusStep_Run(t *testing.T) {
	helmScheme := helmStatusScheme()

	release := &fluxhelm.HelmRelease{
		Namespace: testNamespace,
		Name:      testDoguName,
	}

	t.Run("should requeue if HelmRelease is not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{})

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
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{})

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		require.Error(t, result.Err)
		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to get HelmRelease")
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.False(t, result.Continue)
	})

	t.Run("should requeue while the release is installing", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(release.DeepCopy()).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{})

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
	})

	t.Run("should continue and set Healthy if the desired chart version is deployed and the dogu is healthy", func(t *testing.T) {
		deployed := release.DeepCopy()
		deployed.Generation = 1
		deployed.Status.ObservedGeneration = 1
		deployed.Status.History = fluxhelm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: testVersion + "+ebb48ffcdce3"}}
		doguResource := testDoguResource.DeepCopy()
		c := fake.NewClientBuilder().WithScheme(helmScheme).
			WithObjects(deployed, doguResource).
			WithStatusSubresource(&v3beta1.Dogu{}).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{state: healthyState})

		result := step.Run(context.Background(), doguResource)

		assert.Equal(t, stepsv3.Continue(), result)
		persisted := &v3beta1.Dogu{}
		require.NoError(t, c.Get(context.Background(), testNamespacedName, persisted))
		healthy := meta.FindStatusCondition(persisted.Status.Conditions, v3beta1.ConditionHealthy)
		require.NotNil(t, healthy)
		assert.Equal(t, metav1.ConditionTrue, healthy.Status)
		assert.Equal(t, v3beta1.ReasonSucceeded, healthy.Reason)
	})
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
		{name: "install failed", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseInstallFailed, Message: "msg"}, want: stepsv3.Abort(ReasonInstallFailed, "msg")},
		{name: "upgrade failed", state: fluxstate.ReleaseState{Phase: fluxstate.PhaseUpgradeFailed, Message: "msg", EverDeployed: true}, want: stepsv3.Abort(ReasonUpgradeFailed, "msg")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t), stubHealthChecker{err: assert.AnError})

			got := step.stepResultForRelease(context.Background(), tt.state, testDoguResource.DeepCopy())

			assert.Equal(t, tt.want, got)
		})
	}

	deployedTests := []struct {
		name         string
		everDeployed bool
		checker      stubHealthChecker
		want         stepsv3.StepResult
	}{
		{name: "deployed and healthy", checker: stubHealthChecker{state: healthyState}, want: stepsv3.Continue()},
		{
			name:    "first install deployed but not healthy",
			checker: stubHealthChecker{state: unhealthyState},
			want:    stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "Dogu is not healthy"),
		},
		{
			name:         "upgrade deployed but not healthy",
			everDeployed: true,
			checker:      stubHealthChecker{state: unhealthyState},
			want:         stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonUpgrading, "Dogu is not healthy"),
		},
	}

	for _, tt := range deployedTests {
		t.Run(tt.name, func(t *testing.T) {
			doguResource := testDoguResource.DeepCopy()
			c := fake.NewClientBuilder().WithScheme(helmStatusScheme()).
				WithObjects(doguResource).
				WithStatusSubresource(&v3beta1.Dogu{}).Build()
			step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), tt.checker)
			state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: tt.everDeployed}

			got := step.stepResultForRelease(context.Background(), state, doguResource)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("deployed but health check fails", func(t *testing.T) {
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t), stubHealthChecker{err: assert.AnError})
		state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: true}

		got := step.stepResultForRelease(context.Background(), state, testDoguResource.DeepCopy())

		assert.ErrorIs(t, got.Err, assert.AnError)
		assert.ErrorContains(t, got.Err, "error checking dogu health")
		assert.Equal(t, v3beta1.ReasonUpgrading, got.ReadyReason)
		assert.False(t, got.Continue)
	})

	t.Run("deployed but the status update conflicts", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		c := fake.NewClientBuilder().WithScheme(helmStatusScheme()).
			WithObjects(doguResource).
			WithStatusSubresource(&v3beta1.Dogu{}).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
					return apierrors.NewConflict(v3beta1.GroupVersion.WithResource("dogus").GroupResource(), testDoguName, assert.AnError)
				},
			}).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{state: healthyState})
		state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: true}

		got := step.stepResultForRelease(context.Background(), state, doguResource)

		assert.NoError(t, got.Err)
		assert.Equal(t, defaultRequeueAfter, got.RequeueAfter)
		assert.Equal(t, v3beta1.ReasonUpgrading, got.ReadyReason)
		assert.False(t, got.Continue)
	})

	t.Run("unknown phase", func(t *testing.T) {
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t), stubHealthChecker{})

		got := step.stepResultForRelease(context.Background(), fluxstate.ReleaseState{Phase: "Unknown"}, testDoguResource.DeepCopy())

		assert.ErrorContains(t, got.Err, `unknown release phase "Unknown"`)
		assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
		assert.False(t, got.Continue)
	})
}

func TestHelmReleaseStatusStep_checkHealthy(t *testing.T) {
	t.Run("should set and persist the Healthy condition", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		c := fake.NewClientBuilder().WithScheme(helmStatusScheme()).
			WithObjects(doguResource).
			WithStatusSubresource(&v3beta1.Dogu{}).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{state: unhealthyState})

		healthy, err := step.checkHealthy(context.Background(), doguResource)

		require.NoError(t, err)
		assert.False(t, healthy)
		persisted := &v3beta1.Dogu{}
		require.NoError(t, c.Get(context.Background(), testNamespacedName, persisted))
		condition := meta.FindStatusCondition(persisted.Status.Conditions, v3beta1.ConditionHealthy)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, v3beta1.ReasonWorkloadsNotReady, condition.Reason)
		assert.Equal(t, "workloads not ready", condition.Message)
	})

	t.Run("should not update the status if the Healthy condition did not change", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		doguResource.Status.Conditions = []metav1.Condition{{
			Type: v3beta1.ConditionHealthy, Status: metav1.ConditionTrue, Reason: healthyState.Reason, Message: healthyState.Message,
		}}
		// the mock fails on any unexpected call, e.g. a status update
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), NewMockEventRecorder(t), stubHealthChecker{state: healthyState})

		healthy, err := step.checkHealthy(context.Background(), doguResource)

		require.NoError(t, err)
		assert.True(t, healthy)
	})

	t.Run("should return error if the status cannot be updated", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		c := fake.NewClientBuilder().WithScheme(helmStatusScheme()).
			WithObjects(doguResource).
			WithStatusSubresource(&v3beta1.Dogu{}).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, client client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
					return assert.AnError
				},
			}).Build()
		step := NewHelmReleaseStatusStep(c, NewMockEventRecorder(t), stubHealthChecker{state: healthyState})

		_, err := step.checkHealthy(context.Background(), doguResource)

		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu status resource")
	})
}
