package doguv3

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
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

// These test helpers simulate registry responses and chart operations.
// The test runs the actual lifecycle steps and values assembly against a fake Kubernetes client.
type lifecycleRegistry struct{}

func (lifecycleRegistry) Get(_ context.Context, id doguv3.Identifier) (*doguv3.Dogu, error) {
	return &doguv3.Dogu{Chart: "oci://registry.example/scm", Version: id.Version}, nil
}

type lifecycleChart struct {
	validationError error
	metaError       error
}

func (c *lifecycleChart) DoguMetaValues(context.Context, *v3beta1.Dogu) ([]byte, bool, error) {
	return nil, false, c.metaError
}

func (c *lifecycleChart) ValidateValues(context.Context, *v3beta1.Dogu, map[string]any) error {
	return c.validationError
}

func (lifecycleChart) Render(context.Context, *v3beta1.Dogu, map[string]any) ([]*unstructured.Unstructured, error) {
	return nil, nil
}

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
			Spec:       v3beta1.DoguSpec{Name: "scm", Version: "2.0.0", Stopped: true, Values: apiext.JSON{Raw: []byte(`{"mode":"new"}`)}},
		},
		release: &helm.HelmRelease{
			ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem"},
			Spec:       helm.HelmReleaseSpec{Suspend: true, Values: &apiext.JSON{Raw: []byte(`{"mode":"old"}`)}},
			Status:     helm.HelmReleaseStatus{History: helm.Snapshots{{Version: 1, Status: "deployed", ChartVersion: "1.0.0"}}},
		},
		repository: &source.OCIRepository{
			ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem"},
			Spec:       source.OCIRepositorySpec{URL: "oci://registry.example/scm", Reference: &source.OCIRepositoryRef{Tag: "1.0.0"}},
			Status:     source.OCIRepositoryStatus{Artifact: &fluxmeta.Artifact{}, Conditions: []metav1.Condition{{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue}}},
		},
		deployment: &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem", UID: "web-uid", Labels: map[string]string{v3beta1.DoguLabelName: "scm"}},
			Spec:       apps.DeploymentSpec{Replicas: ptr.To[int32](2)},
		},
		chart: &lifecycleChart{},
	}
	global := &core.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "global-config", Namespace: "ecosystem"}, Data: map[string]string{"config.yaml": "fqdn: ces.example"}}
	f.k8s = fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(f.dogu, f.release).
		WithObjects(f.dogu, f.release, f.repository, f.deployment, global).Build()
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.dogu), f.dogu))
	helmClient := kubernetesfake.NewClientset()
	f.storage = storage.Init(driver.NewSecrets(helmClient.CoreV1().Secrets(f.dogu.Namespace)))
	recorder := record.NewFakeRecorder(100)
	f.useCase = NewDoguInstallOrChangeUseCase(
		install.NewCreateFinalizerStep(f.k8s),
		install.NewEnsureOCIRepositoryStep(f.k8s, lifecycleRegistry{}, recorder),
		install.NewWaitForOCIRepositoryReadyStep(f.k8s, recorder),
		install.NewValidateChartStep(f.chart, values.NewAssembler(f.k8s), f.k8s),
		install.NewEnsureHelmReleaseStep(f.k8s, &config.OperatorConfig{}, f.chart, recorder),
		install.NewHelmReleaseStatusStep(f.k8s, fluxstate.NewHelmReleaseReader(f.k8s, helmClient), recorder), f.k8s, recorder,
	)
	return f
}

func TestV3PendingWithIncompleteSourceComparison(t *testing.T) {
	for _, tc := range []struct {
		name         string
		chartVersion string
		want         metav1.ConditionStatus
	}{
		{name: "matching version is not enough", chartVersion: "2.0.0", want: metav1.ConditionUnknown},
		{name: "known version change remains pending", chartVersion: "1.0.0", want: metav1.ConditionTrue},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.dogu.Spec.PauseReconciliation = true
			require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
			f.release.Status.History[0].ChartVersion = tc.chartVersion
			require.NoError(t, f.k8s.Status().Update(t.Context(), f.release))

			_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

			require.NoError(t, err)
			condition := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
			require.NotNil(t, condition)
			assert.Equal(t, tc.want, condition.Status)
		})
	}
}

func (f *lifecycleFixture) stop(t *testing.T) {
	t.Helper()
	// A second pass confirms that the workloads have stopped after scaling down.
	delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Positive(t, delay)
	delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Zero(t, delay, "an intentional stop must not wait for a rollout")
	require.True(t, f.dogu.Status.Stopped)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
}

func TestV3StoppedReleaseUpdatesInput(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)

	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.Equal(t, "2.0.0", f.release.Labels[v3beta1.DoguLabelVersion])
	assert.JSONEq(t, `{"mode":"new","global":{"cesConfig":{"fqdn":"ces.example"}}}`, string(f.release.Spec.Values.Raw))
	assert.True(t, f.release.Spec.Suspend)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.deployment), f.deployment))
	assert.Zero(t, *f.deployment.Spec.Replicas)
}

func TestV3InvalidValuesPreserveStoppedRelease(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	f.chart.validationError = assert.AnError
	f.dogu.Spec.Values.Raw = []byte(`{"mode":"invalid"}`)
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

	_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

	require.NoError(t, err)
	assert.True(t, f.dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionFalse(f.dogu.Status.Conditions, v3beta1.ConditionValid))
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.Equal(t, "new", f.release.GetValues()["mode"])
}

// recordDeployment simulates Flux's successful Helm action and its stored release revision.
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

func TestV3StartWaitsForResumeAndCurrentHealth(t *testing.T) {
	f := newLifecycleFixture(t)
	f.stop(t)
	// Starting prepares the desired input but waits for the separate suspension step.
	f.dogu.Spec.Stopped = false
	f.dogu.Generation++
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
	delay, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Positive(t, delay)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.True(t, f.release.Spec.Suspend)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.deployment), f.deployment))
	assert.Zero(t, *f.deployment.Spec.Replicas)

	f.release.Spec.Suspend = false
	require.NoError(t, f.k8s.Update(t.Context(), f.release))
	f.recordDeployment(t)
	delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Positive(t, delay, "old stopped status cannot bypass current health")
	meta.SetStatusCondition(&f.dogu.Status.Conditions, metav1.Condition{
		Type: v3beta1.ConditionHealthy, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded,
		Message: "all workloads are ready", ObservedGeneration: f.dogu.Generation,
	})
	require.NoError(t, f.k8s.Status().Update(t.Context(), f.dogu))
	delay, err = f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.Zero(t, delay)
	assert.False(t, f.dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionReady))
}

func TestV3ChangedInputWithIncompleteSourceComparison(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, *lifecycleFixture)
	}{
		{
			name: "global values change without a new dogu generation",
			change: func(t *testing.T, f *lifecycleFixture) {
				global := &core.ConfigMap{}
				require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKey{Namespace: f.dogu.Namespace, Name: "global-config"}, global))
				global.Data["config.yaml"] = "fqdn: changed.example"
				require.NoError(t, f.k8s.Update(t.Context(), global))
			},
		},
		{
			name: "values change during pause without rewriting the release",
			change: func(t *testing.T, f *lifecycleFixture) {
				f.dogu.Spec.PauseReconciliation = true
				f.dogu.Spec.Values.Raw = []byte(`{"mode":"paused"}`)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newLifecycleFixture(t)
			f.stop(t)
			f.recordDeployment(t)
			// An incomplete comparison must replace a previous False, not keep it as confirmed.
			meta.SetStatusCondition(&f.dogu.Status.Conditions, metav1.Condition{
				Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionFalse, Reason: v3beta1.ReasonStopped,
				ObservedGeneration: f.dogu.Generation,
			})
			require.NoError(t, f.k8s.Status().Update(t.Context(), f.dogu))
			generation := f.dogu.Generation
			tc.change(t, f)
			require.NoError(t, f.k8s.Update(t.Context(), f.dogu))

			_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)

			require.NoError(t, err)
			condition := meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
			require.NotNil(t, condition)
			// Until the OCI check is implemented, values cannot be fully compared.
			assert.Equal(t, metav1.ConditionUnknown, condition.Status)
			assert.Equal(t, generation, f.dogu.Generation)
			assert.True(t, f.dogu.Status.Stopped)
			require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
			assert.Equal(t, "new", f.release.GetValues()["mode"])
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

func TestV3PausedReleasePreservesResources(t *testing.T) {
	f := newLifecycleFixture(t)
	f.dogu.Spec.PauseReconciliation = true
	require.NoError(t, f.k8s.Update(t.Context(), f.dogu))
	// Manual support edits must survive, including when stopped is also requested.
	f.release.Spec.Values.Raw = []byte(`{"mode":"support"}`)
	require.NoError(t, f.k8s.Update(t.Context(), f.release))
	_, err := f.useCase.HandleUntilApplied(t.Context(), f.dogu)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(f.dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.Equal(t, v3beta1.ReasonPauseReconciliation, meta.FindStatusCondition(f.dogu.Status.Conditions, v3beta1.ConditionReady).Reason)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.release), f.release))
	assert.JSONEq(t, `{"mode":"support"}`, string(f.release.Spec.Values.Raw))
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.repository), f.repository))
	assert.Equal(t, "1.0.0", f.repository.Spec.Reference.Tag)
	require.NoError(t, f.k8s.Get(t.Context(), client.ObjectKeyFromObject(f.deployment), f.deployment))
	assert.Equal(t, int32(2), *f.deployment.Spec.Replicas)
	assert.Empty(t, f.dogu.Finalizers, "pause must precede normal writes")
	assert.False(t, f.dogu.Status.Stopped)
}
