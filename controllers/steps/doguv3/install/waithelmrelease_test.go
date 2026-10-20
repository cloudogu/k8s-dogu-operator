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
	sdk "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
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

	step := NewHelmReleaseStatusStep(k8sClient, nil, recorder)

	assert.NotNil(t, step)
	assert.Equal(t, k8sClient, step.k8sClient)
	assert.Equal(t, recorder, step.eventRecorder)
}

func TestHelmReleaseStatusStep_PendingChange(t *testing.T) {
	for _, tc := range []struct {
		name       string
		missing    bool
		getErr     error
		version    string
		storedMode string
		want       metav1.ConditionStatus
		wantErr    string
	}{
		{name: "missing release needs installation", missing: true, want: metav1.ConditionTrue},
		{name: "read failure cannot confirm deployment", getErr: assert.AnError, want: metav1.ConditionUnknown, wantErr: "failed to read HelmRelease"},
		{name: "version change is known without a stored release", version: "1.0.0", want: metav1.ConditionTrue},
		{name: "missing Helm secret cannot confirm deployment", version: "2.0.0", want: metav1.ConditionUnknown, wantErr: "failed to read stored Helm release"},
		{name: "stored values differ despite current HelmRelease input", version: "2.0.0", storedMode: "old", want: metav1.ConditionTrue},
		{name: "matching stored revision confirms deployment", version: "2.0.0", storedMode: "new", want: metav1.ConditionFalse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dogu := &v3beta1.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem"},
				Spec:       v3beta1.DoguSpec{Name: "scm", Version: "2.0.0"},
			}
			desired := &fluxhelm.HelmRelease{
				Spec: fluxhelm.HelmReleaseSpec{Values: &apiext.JSON{Raw: []byte(`{"mode":"new"}`)}},
			}
			actual := desired.DeepCopy()
			actual.Name, actual.Namespace = dogu.Spec.Name, dogu.Namespace
			// The comparison must use the revision referenced by Flux, not just the CR's current values.
			actual.Status = fluxhelm.HelmReleaseStatus{
				History:    fluxhelm.Snapshots{{Name: "installed-scm", Version: 3, ChartVersion: tc.version, Status: "deployed"}},
				Conditions: []metav1.Condition{{Type: fluxhelm.ReleasedCondition, Status: metav1.ConditionTrue}},
			}
			builder := fake.NewClientBuilder().WithScheme(helmStatusScheme())
			if !tc.missing {
				builder.WithObjects(actual)
			}
			if tc.getErr != nil {
				builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return tc.getErr
					},
				})
			}
			k8s := builder.Build()
			helmClient := kubernetesfake.NewClientset()
			if tc.storedMode != "" {
				store := storage.Init(driver.NewSecrets(helmClient.CoreV1().Secrets(dogu.Namespace)))
				require.NoError(t, store.Create(&sdk.Release{
					Name: "installed-scm", Namespace: dogu.Namespace, Version: 3,
					Config: map[string]any{"mode": tc.storedMode}, Info: &sdk.Info{Status: sdk.StatusDeployed},
				}))
			}
			step := NewHelmReleaseStatusStep(k8s, fluxstate.NewHelmReleaseReader(k8s, helmClient), nil)

			change, err := step.PendingChange(t.Context(), dogu, desired)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			if tc.getErr != nil {
				assert.ErrorIs(t, err, tc.getErr)
			}
			assert.Equal(t, tc.want, change.Status)
		})
	}
}

func TestHelmReleaseStatusStep_Run(t *testing.T) {
	helmScheme := helmStatusScheme()

	release := &fluxhelm.HelmRelease{
		Namespace: testNamespace,
		Name:      testDoguName,
	}

	t.Run("should requeue if HelmRelease is not found", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).Build()
		step := NewHelmReleaseStatusStep(c, nil, NewMockEventRecorder(t))

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
		step := NewHelmReleaseStatusStep(c, nil, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		require.Error(t, result.Err)
		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to get HelmRelease")
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.False(t, result.Continue)
	})

	t.Run("should requeue while the release is installing", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(release.DeepCopy()).Build()
		step := NewHelmReleaseStatusStep(c, nil, NewMockEventRecorder(t))

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
		step := NewHelmReleaseStatusStep(c, nil, NewMockEventRecorder(t))

		result := step.Run(context.Background(), doguResource)

		assert.Equal(t, stepsv3.Continue(), result)
	})
	t.Run("old stopped status and stale healthy condition do not complete a start", func(t *testing.T) {
		deployed := release.DeepCopy()
		deployed.Status.History = fluxhelm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: testVersion}}
		dogu := testDoguResource.DeepCopy()
		dogu.Generation = 2
		dogu.Status.Stopped = true
		dogu.Status.Conditions = []metav1.Condition{healthyCondition(metav1.ConditionTrue, "old health")}
		dogu.Status.Conditions[0].ObservedGeneration = 1
		k8s := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(deployed).Build()
		result := NewHelmReleaseStatusStep(k8s, nil, nil).Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.False(t, result.Continue)
		assert.Positive(t, result.RequeueAfter)
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
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, NewMockEventRecorder(t))

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
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, NewMockEventRecorder(t))
			state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: tt.everDeployed}

			got := step.stepResultForRelease(state, doguResource)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("unknown phase", func(t *testing.T) {
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, NewMockEventRecorder(t))

		got := step.stepResultForRelease(fluxstate.ReleaseState{Phase: "Unknown"}, testDoguResource.DeepCopy())

		assert.ErrorContains(t, got.Err, `unknown release phase "Unknown"`)
		assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
		assert.False(t, got.Continue)
	})
}
