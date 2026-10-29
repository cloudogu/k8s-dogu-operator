package install

import (
	"context"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	sdk "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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

	step := NewHelmReleaseStatusStep(k8sClient, nil, nil, &config.OperatorConfig{}, nil, recorder)

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
		step := NewHelmReleaseStatusStep(c, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

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
		step := NewHelmReleaseStatusStep(c, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

		result := step.Run(context.Background(), testDoguResource.DeepCopy())

		require.Error(t, result.Err)
		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, "failed to get HelmRelease")
		assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
		assert.False(t, result.Continue)
	})

	t.Run("should requeue while the release is installing", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithObjects(release.DeepCopy()).Build()
		step := NewHelmReleaseStatusStep(c, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

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
		step := NewHelmReleaseStatusStep(c, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

		result := step.Run(context.Background(), doguResource)

		assert.Equal(t, stepsv3.Continue(), result)
	})
	t.Run("should report pending installation during pause", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		doguResource.Spec.PauseReconciliation = true
		c := fake.NewClientBuilder().WithScheme(helmScheme).WithStatusSubresource(doguResource).WithObjects(doguResource).Build()
		step := NewHelmReleaseStatusStep(c, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

		result := step.Run(context.Background(), doguResource)

		assert.Equal(t, stepsv3.Continue(), result)
		require.Len(t, doguResource.Status.Conditions, 1)
		assert.Equal(t, v3beta1.ConditionUpdatePending, doguResource.Status.Conditions[0].Type)
		assert.Equal(t, metav1.ConditionTrue, doguResource.Status.Conditions[0].Status)
	})

	for _, tt := range []struct {
		name          string
		stopRequested bool
		stopCompleted bool
		suspended     bool
		wantReason    string
	}{
		{name: "running dogu waits for unsuspension", suspended: true, wantReason: v3beta1.ReasonStarting},
		{name: "requested stop waits for suspension", stopRequested: true, wantReason: v3beta1.ReasonStopping},
		{
			name:          "suspension alone does not confirm workload stop",
			stopRequested: true,
			suspended:     true,
			wantReason:    v3beta1.ReasonStarting,
		},
		{
			name:          "completed stop continues while suspended",
			stopRequested: true,
			stopCompleted: true,
			suspended:     true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dogu := testDoguResource.DeepCopy()
			dogu.Spec.Stopped = tt.stopRequested
			dogu.Status.Stopped = tt.stopCompleted
			suspendedRelease := release.DeepCopy()
			suspendedRelease.Spec.Suspend = tt.suspended
			k8s := fake.NewClientBuilder().WithScheme(helmScheme).WithStatusSubresource(dogu).
				WithObjects(dogu, suspendedRelease).Build()
			step := NewHelmReleaseStatusStep(k8s, nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

			result := step.Run(t.Context(), dogu)

			require.NoError(t, result.Err)
			if tt.wantReason == "" {
				assert.Equal(t, stepsv3.Continue(), result)
			} else {
				assert.False(t, result.Continue)
				assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
				assert.Equal(t, tt.wantReason, result.ReadyReason)
			}
		})
	}
}

func healthyCondition(status metav1.ConditionStatus, message string) metav1.Condition {
	return metav1.Condition{Type: v3beta1.ConditionHealthy, Status: status, Reason: "SomeReason", Message: message}
}

func TestHelmReleaseStatusStep_PendingChanges(t *testing.T) {
	const deployedManifest = `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: scm-config
  labels:
    app: ces
    k8s.cloudogu.com/app: ces
    dogu.name: scm
    k8s.cloudogu.com/dogu.name: scm
    app.kubernetes.io/name: scm
    app.kubernetes.io/version: 2.0.0
    app.kubernetes.io/part-of: ces
`

	tests := []struct {
		name           string
		change         func(*v3beta1.Dogu, *fluxhelm.HelmRelease, *sdk.Release)
		missingStorage bool
		metadataError  bool
		wantPending    bool
		wantRetry      bool
		wantError      string
	}{
		{name: "matching prepared and deployed inputs"},
		{
			name: "chart version has not been deployed",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Status.History[0].ChartVersion = "1.0.0"
			},
			wantPending: true,
		},
		{
			name: "release has no installation history",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Status.History = nil
			},
			wantPending: true,
		},
		{
			name: "prepared values differ from desired values",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Spec.Values.Raw = []byte(`{"replicas":3}`)
			},
			wantPending: true,
		},
		{
			name: "prepared values match but deployed values are outdated",
			change: func(_ *v3beta1.Dogu, _ *fluxhelm.HelmRelease, stored *sdk.Release) {
				stored.Config["replicas"] = 3
			},
			wantPending: true,
		},
		{
			name: "numeric strings are not equivalent to numeric values",
			change: func(_ *v3beta1.Dogu, _ *fluxhelm.HelmRelease, stored *sdk.Release) {
				stored.Config["replicas"] = "2"
			},
			wantPending: true,
		},
		{
			name: "chart reference differs",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Spec.ChartRef.Name = "other-source"
			},
			wantPending: true,
		},
		{
			name: "schema validation policy differs",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Spec.Install.DisableSchemaValidation = true
			},
			wantPending: true,
		},
		{
			name: "prepared metadata differs",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Spec.CommonMetadata.Labels[v3beta1.DoguLabelName] = "other-dogu"
			},
			wantPending: true,
		},
		{
			name: "every stored resource must have the desired metadata",
			change: func(_ *v3beta1.Dogu, _ *fluxhelm.HelmRelease, stored *sdk.Release) {
				stored.Manifest += "\n---\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: missing-labels\n"
			},
			wantPending: true,
		},
		{
			name: "invalid stored manifest prevents resetting pending",
			change: func(_ *v3beta1.Dogu, _ *fluxhelm.HelmRelease, stored *sdk.Release) {
				stored.Manifest = "apiVersion: ["
			},
			wantError: "failed to decode stored manifest for HelmRelease ecosystem/scm",
		},
		{
			name: "failed latest revision cannot confirm applied inputs",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Status.History[0].Status = string(sdk.StatusFailed)
			},
			wantRetry: true,
		},
		{
			name: "missing release confirmation prevents resetting pending",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Status.Conditions = nil
			},
			wantRetry: true,
		},
		{
			name: "acknowledged suspension permits a stale Reconciling condition",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				meta.SetStatusCondition(&release.Status.Conditions, metav1.Condition{
					Type: fluxmeta.ReconcilingCondition, Status: metav1.ConditionTrue, Reason: fluxmeta.ProgressingReason, ObservedGeneration: 7,
				})
			},
		},
		{
			name: "unacknowledged suspension still waits for reconciliation",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Status.ObservedGeneration = 7
				meta.SetStatusCondition(&release.Status.Conditions, metav1.Condition{
					Type: fluxmeta.ReconcilingCondition, Status: metav1.ConditionTrue, Reason: fluxmeta.ProgressingReason, ObservedGeneration: 7,
				})
			},
			wantRetry: true,
		},
		{
			name: "unsuspended release still waits for reconciliation",
			change: func(_ *v3beta1.Dogu, release *fluxhelm.HelmRelease, _ *sdk.Release) {
				release.Spec.Suspend = false
				meta.SetStatusCondition(&release.Status.Conditions, metav1.Condition{
					Type: fluxmeta.ReconcilingCondition, Status: metav1.ConditionTrue, Reason: fluxmeta.ProgressingReason, ObservedGeneration: 8,
				})
			},
			wantRetry: true,
		},
		{
			name: "known current-generation pending survives unavailable chart metadata",
			change: func(dogu *v3beta1.Dogu, _ *fluxhelm.HelmRelease, _ *sdk.Release) {
				meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
					Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionTrue,
					Reason: v3beta1.ReasonReconciliationPaused, ObservedGeneration: dogu.Generation,
				})
			},
			metadataError: true,
			wantPending:   true,
		},
		{
			name:           "missing stored revision reports an error",
			missingStorage: true,
			wantError:      "failed to read stored Helm release",
		},
		{
			name:          "unavailable chart metadata reports an error",
			metadataError: true,
			wantError:     "failed to retrieve dogu metadata values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			dogu := &v3beta1.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 4},
				Spec:       v3beta1.DoguSpec{Name: "scm", Version: "2.0.0", PauseReconciliation: true},
			}

			release := &fluxhelm.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: dogu.Spec.Name, Namespace: dogu.Namespace, Generation: 8}}
			configureHelmRelease(release, dogu, &apiext.JSON{Raw: []byte(`{"replicas":2}`)}, &metav1.Duration{}, &metav1.Duration{})
			release.Status = fluxhelm.HelmReleaseStatus{
				ObservedGeneration: 8,
				History:            fluxhelm.Snapshots{{Name: "installed-scm", Version: 3, ChartVersion: "2.0.0+artifact-digest", Status: "deployed"}},
				Conditions:         []metav1.Condition{{Type: fluxhelm.ReleasedCondition, Status: metav1.ConditionTrue, Reason: "UpgradeSucceeded"}},
			}

			storedRelease := &sdk.Release{
				Name:      release.Status.History[0].Name,
				Namespace: dogu.Namespace,
				Version:   release.Status.History[0].Version,
				Info:      &sdk.Info{Status: sdk.StatusDeployed},
				Config:    map[string]any{"replicas": 2},
				Manifest:  deployedManifest,
			}
			if tt.change != nil {
				tt.change(dogu, release, storedRelease)
			}

			k8sClient := fake.NewClientBuilder().WithScheme(helmStatusScheme()).WithStatusSubresource(dogu).
				WithObjects(dogu, release).Build()
			helmClient := kubernetesfake.NewClientset()
			if !tt.missingStorage {
				require.NoError(t, storage.Init(driver.NewSecrets(helmClient.CoreV1().Secrets(dogu.Namespace))).Create(storedRelease))
			}

			chartService := newMockChartService(t)
			assembler := newMockValueAssembler(t)
			var metadataError error
			if tt.metadataError {
				metadataError = assert.AnError
			}
			chartService.EXPECT().DoguMetaValues(mock.Anything, mock.Anything).Return(nil, false, metadataError).Maybe()
			chartService.EXPECT().RuntimePatchTemplate(mock.Anything, mock.Anything).Return(nil, false, nil).Maybe()
			assembler.EXPECT().Assemble(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(values.Values{"replicas": 2}, nil).Maybe()
			step := NewHelmReleaseStatusStep(k8sClient, chartService, assembler, &config.OperatorConfig{},
				fluxstate.NewHelmReleaseReader(k8sClient, helmClient), NewMockEventRecorder(t))

			result := step.Run(ctx, dogu)

			if tt.wantError != "" {
				require.ErrorContains(t, result.Err, tt.wantError)
				assert.False(t, result.Continue)
				return
			}
			require.NoError(t, result.Err)
			if tt.wantRetry {
				assert.False(t, result.Continue)
				assert.Equal(t, defaultRequeueAfter, result.RequeueAfter)
				assert.Equal(t, v3beta1.ReasonInstalling, result.ReadyReason)
			} else {
				assert.Equal(t, stepsv3.Continue(), result)
			}

			require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(dogu), dogu))
			pending := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
			if tt.wantPending {
				require.NotNil(t, pending)
				assert.Equal(t, metav1.ConditionTrue, pending.Status)
				assert.Equal(t, dogu.Generation, pending.ObservedGeneration)
				assert.Equal(t, v3beta1.ReasonReconciliationPaused, pending.Reason)
			} else {
				assert.Nil(t, pending, "this step must not write False or report an unproven difference")
			}
		})
	}
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
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

			got := step.stepResultForRelease(testCtx, tt.state, testDoguResource.DeepCopy())

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
			step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))
			state := fluxstate.ReleaseState{Phase: fluxstate.PhaseDeployed, EverDeployed: tt.everDeployed}

			got := step.stepResultForRelease(testCtx, state, doguResource)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("unknown phase", func(t *testing.T) {
		step := NewHelmReleaseStatusStep(NewMockK8sClient(t), nil, nil, &config.OperatorConfig{}, nil, NewMockEventRecorder(t))

		got := step.stepResultForRelease(testCtx, fluxstate.ReleaseState{Phase: "Unknown"}, testDoguResource.DeepCopy())

		assert.ErrorContains(t, got.Err, `unknown release phase "Unknown"`)
		assert.Equal(t, v3beta1.ReasonInstalling, got.ReadyReason)
		assert.False(t, got.Continue)
	})
}
