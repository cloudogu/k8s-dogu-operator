package doguv3

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	helm "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	source "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type lifecycleRegistry struct{}

func (lifecycleRegistry) Get(_ context.Context, id doguv3.Identifier) (*doguv3.Dogu, error) {
	return &doguv3.Dogu{Chart: "oci://registry.example/scm", Version: id.Version}, nil
}

type lifecycleChart struct {
	metaError            error
	runtimePatchTemplate []byte
}

func (c *lifecycleChart) DoguMetaValues(context.Context, *v3beta1.Dogu) ([]byte, bool, error) {
	return nil, false, c.metaError
}

func (c *lifecycleChart) RuntimePatchTemplate(context.Context, *v3beta1.Dogu) ([]byte, bool, error) {
	return c.runtimePatchTemplate, c.runtimePatchTemplate != nil, nil
}

func (*lifecycleChart) ValidateValues(context.Context, *v3beta1.Dogu, map[string]any) error {
	return nil
}

func (*lifecycleChart) Render(context.Context, *v3beta1.Dogu, map[string]any) ([]*unstructured.Unstructured, error) {
	return nil, nil
}

// These tests run the normal chain with real values assembly and fake Kubernetes
// and Helm storage. Chart rendering and Flux actions are simulated, not live.
type lifecycleFixture struct {
	dogu       *v3beta1.Dogu
	release    *helm.HelmRelease
	repository *source.OCIRepository
	deployment *apps.Deployment
	chart      *lifecycleChart
	k8s        client.Client
	storage    *storage.Storage
	useCase    *DoguUseCase
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{v3beta1.AddToScheme, apps.AddToScheme, core.AddToScheme, helm.AddToScheme, source.AddToScheme} {
		require.NoError(t, add(scheme))
	}
	f := &lifecycleFixture{
		dogu: &v3beta1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", UID: "dogu-uid", Generation: 2},
			Spec: v3beta1.DoguSpec{Name: "scm", Version: "2.0.0", Stopped: true,
				Values: apiext.JSON{Raw: []byte(`{"mode":"new"}`)}},
		},
		release: &helm.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem"},
			Spec:       helm.HelmReleaseSpec{Suspend: true, Values: &apiext.JSON{Raw: []byte(`{"mode":"old"}`)}},
			Status:     helm.HelmReleaseStatus{History: helm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: "1.0.0"}}},
		},
		repository: &source.OCIRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem"},
			Spec:       source.OCIRepositorySpec{URL: "oci://registry.example/scm", Reference: &source.OCIRepositoryRef{Tag: "1.0.0"}},
			Status: source.OCIRepositoryStatus{Artifact: &fluxmeta.Artifact{},
				Conditions: []metav1.Condition{{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue}}},
		},
		deployment: &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem", UID: "web-uid", Labels: map[string]string{v3beta1.DoguLabelName: "scm"}},
			Spec:       apps.DeploymentSpec{Replicas: ptr.To[int32](2)},
		},
		chart: &lifecycleChart{},
	}
	global := &core.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "global-config", Namespace: "ecosystem"},
		Data: map[string]string{"config.yaml": "fqdn: ces.example"}}
	f.k8s = fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(f.dogu, f.release).
		WithObjects(f.dogu, f.release, f.repository, f.deployment, global).Build()
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), f.dogu))
	helmClient := kubernetesfake.NewClientset()
	f.storage = storage.Init(driver.NewSecrets(helmClient.CoreV1().Secrets(f.dogu.Namespace)))
	recorder := record.NewFakeRecorder(100)
	assembler := values.NewAssembler(f.k8s)
	operatorConfig := &config.OperatorConfig{}
	f.useCase = NewDoguInstallOrChangeUseCase(
		install.NewCreateFinalizerStep(f.k8s),
		install.NewEnsureOCIRepositoryStep(f.k8s, lifecycleRegistry{}, recorder),
		install.NewWaitForOCIRepositoryReadyStep(f.k8s, recorder),
		install.NewValidateChartStep(f.chart, assembler, f.k8s),
		install.NewSuspendHelmReleaseStep(f.k8s, f.chart, assembler, pvc.NewResizeChecker(f.k8s), recorder),
		install.NewEnsureHelmReleaseStep(f.k8s, operatorConfig, f.chart, assembler, recorder),
		install.NewHelmReleaseStatusStep(f.k8s, f.chart, assembler, operatorConfig, fluxstate.NewHelmReleaseReader(f.k8s, helmClient), recorder),
		install.NewUpdatePendingStep(f.k8s), install.NewExportModeStep(f.k8s, recorder), f.k8s, recorder,
	)
	return f
}

func (f *lifecycleFixture) stop(t *testing.T) {
	t.Helper()
	delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Positive(t, delay)
	delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Zero(t, delay)
	require.True(t, f.dogu.Status.Stopped)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
}

func TestV3StoppedReleaseUpdatesInputAndRetainsPending(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.Equal(t, "2.0.0", f.release.Labels[v3beta1.DoguLabelVersion])
	assert.JSONEq(t, `{"mode":"new","global":{"cesConfig":{"fqdn":"ces.example"}}}`, string(f.release.Spec.Values.Raw))
	assert.True(t, f.release.Spec.Suspend)
	assert.Equal(t, "1.0.0", f.release.Status.History.Latest().ChartVersion)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.deployment), f.deployment))
	assert.Zero(t, *f.deployment.Spec.Replicas)
}

func (f *lifecycleFixture) recordDeployment(t *testing.T) {
	t.Helper()
	f.release.Status.History = helm.Snapshots{{Name: "scm", Version: 2, ChartVersion: "2.0.0", Status: "deployed"}}
	f.release.Status.ObservedGeneration = f.release.Generation
	manifest, err := json.Marshal(&core.ConfigMap{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{Name: "scm-config", Labels: f.release.Spec.CommonMetadata.Labels},
	})
	require.NoError(t, err)
	require.NoError(t, f.storage.Create(&helmrelease.Release{
		Name: "scm", Namespace: f.dogu.Namespace, Version: 2, Config: f.release.GetValues(),
		Info: &helmrelease.Info{Status: helmrelease.StatusDeployed}, Manifest: string(manifest),
	}))
	f.release.Status.Conditions = []metav1.Condition{{Type: helm.ReleasedCondition, Status: metav1.ConditionTrue}}
	require.NoError(t, f.k8s.Status().Update(t.Context(), f.release))
}

func TestV3StoppedReleaseClearsPendingAfterAcknowledgedSuspension(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	f.release.Generation = 8
	require.NoError(t, f.k8s.Update(t.Context(), f.release))
	f.recordDeployment(t)
	f.release.Status.Conditions = append(f.release.Status.Conditions, metav1.Condition{
		Type: fluxmeta.ReconcilingCondition, Status: metav1.ConditionTrue,
		Reason: fluxmeta.ProgressingReason, Message: "Fulfilling prerequisites", ObservedGeneration: 7,
	})
	require.NoError(t, f.k8s.Status().Update(t.Context(), f.release))
	f.dogu.Generation++
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

	delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

	require.NoError(t, err)
	assert.Zero(t, delay, "acknowledged suspension must not retry because of a stale Reconciling condition")
	assert.True(t, f.dogu.Status.Stopped)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), f.dogu))
	pending := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	require.NotNil(t, pending)
	assert.Equal(t, metav1.ConditionFalse, pending.Status)
	assert.Equal(t, f.dogu.Generation, pending.ObservedGeneration)
}

func TestV3PausePreservesResourcesAndCombinesDetection(t *testing.T) {
	f := newLifecycleFixture(t)
	f.dogu.Spec.PauseReconciliation = true
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
	_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.Equal(t, v3beta1.ReasonReconciliationPaused, meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionReady).Reason)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.Equal(t, "old", f.release.GetValues()["mode"])
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.repository), f.repository))
	assert.Equal(t, "1.0.0", f.repository.Spec.Reference.Tag)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.deployment), f.deployment))
	assert.Equal(t, int32(2), *f.deployment.Spec.Replicas)
	assert.Empty(t, f.dogu.Finalizers)
	assert.False(t, f.dogu.Status.Stopped)
}

func TestV3PausedValuesAndRevertedChange(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	f.recordDeployment(t)
	f.dogu.Spec.PauseReconciliation = true
	f.dogu.Spec.Values.Raw = []byte(`{"mode":"paused"}`)
	f.dogu.Generation++
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

	_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

	require.NoError(t, err)
	pending := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	require.NotNil(t, pending)
	assert.Equal(t, metav1.ConditionTrue, pending.Status)
	assert.Equal(t, f.dogu.Generation, pending.ObservedGeneration)
	assert.Equal(t, v3beta1.ReasonReconciliationPaused, pending.Reason, "Helm detection follows an unchanged OCI comparison")

	f.dogu.Spec.Values.Raw = []byte(`{"mode":"new"}`)
	f.dogu.Generation++
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

	_, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)

	require.NoError(t, err)
	pending = meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	assert.Equal(t, metav1.ConditionFalse, pending.Status)
	assert.Equal(t, f.dogu.Generation, pending.ObservedGeneration)
}

func TestV3PausedInvalidStoredManifestReportsComparisonError(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	f.recordDeployment(t)
	stored, err := f.storage.Get("scm", 2)
	require.NoError(t, err)
	stored.Manifest = "apiVersion: ["
	require.NoError(t, f.storage.Update(stored))
	f.dogu.Spec.PauseReconciliation = true
	f.dogu.Generation++
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

	_, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)

	require.ErrorContains(t, err, "failed to decode stored manifest for HelmRelease ecosystem/scm")
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending), "a failed comparison must not clear pending")
}

func TestV3StartWaitsForCurrentHealthBeforeReset(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		stopCompleted bool
	}{
		{name: "start after completed stop", stopCompleted: true},
		{name: "cancel partial stop", stopCompleted: false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newLifecycleFixture(t)
			if scenario.stopCompleted {
				f.stop(t)
			} else {
				delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

				require.NoError(t, err)
				require.Positive(t, delay)
				require.False(t, f.dogu.Status.Stopped)
				require.Equal(t, v3beta1.ReasonPartiallyStopped, meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionStopped).Reason)
			}

			// Request a start; the release must first apply the prepared inputs.
			f.dogu.Spec.Stopped = false
			f.dogu.Generation++
			require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

			delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

			require.NoError(t, err)
			assert.Positive(t, delay)

			// A completed deployment with health from an older generation is not enough.
			require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
			f.recordDeployment(t)
			meta.SetStatusCondition(&f.dogu.Status.Conditions, metav1.Condition{
				Type: v3beta1.ConditionHealthy, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded,
				Message: "old health", ObservedGeneration: f.dogu.Generation - 1,
			})
			require.NoError(t, f.k8s.Status().Update(t.Context(), f.dogu))

			delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)

			require.NoError(t, err)
			assert.Positive(t, delay)
			assert.Equal(t, scenario.stopCompleted, f.dogu.Status.Stopped)
			assert.Equal(t, scenario.stopCompleted, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))

			// Only current-generation health confirms the start and clears pending.
			meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionHealthy).ObservedGeneration = f.dogu.Generation
			require.NoError(t, f.k8s.Status().Update(t.Context(), f.dogu))

			delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)

			require.NoError(t, err)
			assert.Zero(t, delay)
			assert.False(t, f.dogu.Status.Stopped)
			assert.True(t, meta.IsStatusConditionFalse(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))

			require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), f.dogu))
			stopped := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionStopped)
			require.NotNil(t, stopped)
			assert.Equal(t, metav1.ConditionFalse, stopped.Status)
			assert.Equal(t, v3beta1.ReasonNotStopped, stopped.Reason)
			assert.Equal(t, f.dogu.Generation, stopped.ObservedGeneration)
		})
	}
}

func TestV3InitiallyStoppedRelease(t *testing.T) {
	f := newLifecycleFixture(t)
	require.NoError(t, f.k8s.Delete(t.Context(), f.release))
	require.NoError(t, f.k8s.Delete(t.Context(), f.deployment))
	delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Zero(t, delay)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.True(t, f.release.Spec.Suspend)
	assert.True(t, f.dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
}

func TestV3PendingUsesStoredReleaseAndPreservesIncompleteResults(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*helmrelease.Release)
		wait   bool
	}{
		{name: "stored values differ from the prepared input", change: func(stored *helmrelease.Release) {
			stored.Config["mode"] = "old"
		}},
		{name: "stored metadata differs", change: func(stored *helmrelease.Release) {
			stored.Manifest = "apiVersion: v1\nkind: Service\nmetadata:\n  name: scm\n"
		}},
		{name: "failed stored action cannot confirm deployment", wait: true, change: func(stored *helmrelease.Release) {
			stored.Info.Status = helmrelease.StatusFailed
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.stop(t)
			f.recordDeployment(t)
			stored, err := f.storage.Get("scm", 2)
			require.NoError(t, err)
			tc.change(stored)
			require.NoError(t, f.storage.Update(stored))
			// A new generation must be checked instead of retaining an old True.
			previousGeneration := f.dogu.Generation
			f.dogu.Generation++
			f.dogu.Spec.PauseReconciliation = true
			require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
			delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
			require.NoError(t, err)
			pending := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
			require.NotNil(t, pending)
			assert.Equal(t, metav1.ConditionTrue, pending.Status)
			if tc.wait {
				assert.Positive(t, delay, "an incomplete comparison must not reach the reset step")
				assert.Equal(t, previousGeneration, pending.ObservedGeneration)
			} else {
				assert.Zero(t, delay)
				assert.Equal(t, f.dogu.Generation, pending.ObservedGeneration)
			}
		})
	}
}

func TestV3MissingStoredRevisionDoesNotClearPending(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	f.recordDeployment(t)
	_, err := f.storage.Delete("scm", 2)
	require.NoError(t, err)
	f.dogu.Generation++
	f.dogu.Spec.PauseReconciliation = true
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
	_, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.ErrorContains(t, err, "failed to read stored Helm release")
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
}

func TestV3GlobalValuesChangeWithoutNewGeneration(t *testing.T) {
	f := newLifecycleFixture(t)
	f.chart.runtimePatchTemplate = []byte(`runtimeHost: '{{ globalConfig "fqdn" }}'`)
	f.stop(t)
	assert.Equal(t, "ces.example", f.release.GetValues()["runtimeHost"])
	f.recordDeployment(t)
	meta.SetStatusCondition(&f.dogu.Status.Conditions, metav1.Condition{
		Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionFalse,
		Reason: v3beta1.ReasonStopped, ObservedGeneration: f.dogu.Generation,
	})
	require.NoError(t, f.k8s.Status().Update(t.Context(), f.dogu))
	global := &core.ConfigMap{}
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKey{Namespace: f.dogu.Namespace, Name: "global-config"}, global))
	global.Data["config.yaml"] = "fqdn: changed.example"
	require.NoError(t, f.k8s.Update(t.Context(), global))
	_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.Equal(t, int64(2), f.dogu.Generation)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.Equal(t, "changed.example", f.release.GetValues()["runtimeHost"])
	assert.True(t, f.release.Spec.Suspend, "runtime updates must not resume a stopped dogu")
}
