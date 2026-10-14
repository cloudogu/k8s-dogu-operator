package doguv3

import (
	"context"
	"testing"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	helm "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	source "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

func TestV3StoppedReleaseLifecycle(t *testing.T) {
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{v3beta1.AddToScheme, apps.AddToScheme, core.AddToScheme, helm.AddToScheme, source.AddToScheme} {
		require.NoError(t, add(scheme))
	}
	dogu := &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", UID: "dogu-uid", Generation: 2},
		Spec:       v3beta1.DoguSpec{Name: "scm", Version: "2.0.0", Stopped: true, Values: apiext.JSON{Raw: []byte(`{"mode":"new"}`)}},
	}
	release := &helm.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: dogu.Namespace, Labels: map[string]string{v3beta1.DoguLabelVersion: "1.0.0"}},
		Spec:       helm.HelmReleaseSpec{Suspend: true, Values: &apiext.JSON{Raw: []byte(`{"mode":"old"}`)}},
	}
	repository := &source.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: dogu.Namespace},
		Status:     source.OCIRepositoryStatus{Conditions: []metav1.Condition{{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue}}},
	}
	deployment := &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: dogu.Namespace, UID: "web-uid", Labels: map[string]string{v3beta1.DoguLabelName: "scm"}},
		Spec:       apps.DeploymentSpec{Replicas: ptr.To[int32](2)},
	}
	global := &core.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "global-config", Namespace: dogu.Namespace}, Data: map[string]string{"config.yaml": "fqdn: ces.example"}}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dogu, release).
		WithObjects(dogu, release, repository, deployment, global).Build()
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	recorder := record.NewFakeRecorder(30)
	chart := &lifecycleChart{}
	useCase := NewDoguInstallOrChangeUseCase(
		install.NewCreateFinalizerStep(k8s),
		install.NewEnsureOCIRepositoryStep(k8s, lifecycleRegistry{}, recorder),
		install.NewWaitForOCIRepositoryReadyStep(k8s, recorder),
		install.NewValidateChartStep(chart, values.NewAssembler(k8s), k8s),
		install.NewEnsureHelmReleaseStep(k8s, &config.OperatorConfig{}, chart, recorder),
		install.NewHelmReleaseStatusStep(k8s, recorder), k8s, recorder,
	)

	// First stop pass requests replica zero; the second confirms the live state.
	delay, err := useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.Positive(t, delay)
	delay, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.Zero(t, delay, "a stopped dogu must not wait for the deferred upgrade")
	assert.True(t, dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(release), release))
	assert.Equal(t, "1.0.0", release.Labels[v3beta1.DoguLabelVersion])
	assert.JSONEq(t, `{"mode":"old"}`, string(release.Spec.Values.Raw))
	assert.True(t, release.Spec.Suspend)

	// Invalid desired values remain pending and do not prevent the completed stop.
	chart.validationError = assert.AnError
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.True(t, dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionValid))

	// A metadata failure must not erase an already known pending version change.
	chart.metaError = assert.AnError
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.Error(t, err)
	assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	chart.metaError = nil

	// Starting prepares current input but never releases Flux itself.
	chart.validationError = nil
	dogu.Spec.Stopped = false
	dogu.Generation++
	require.NoError(t, k8s.Update(t.Context(), dogu))
	delay, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.Positive(t, delay)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(release), release))
	assert.True(t, release.Spec.Suspend)
	assert.Equal(t, "2.0.0", release.Labels[v3beta1.DoguLabelVersion])
	assert.JSONEq(t, `{"mode":"new","global":{"cesConfig":{"fqdn":"ces.example"}}}`, string(release.Spec.Values.Raw))
	assert.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionReady))
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), deployment))
	assert.Zero(t, *deployment.Spec.Replicas, "the operator must not manually start workloads")

	// Simulate #62934 releasing and Flux applying the prepared release.
	release.Spec.Suspend = false
	require.NoError(t, k8s.Update(t.Context(), release))
	release.Status.History = helm.Snapshots{{Version: 2, ChartVersion: "2.0.0", Status: "deployed"}}
	release.Status.ObservedGeneration = release.Generation
	require.NoError(t, k8s.Status().Update(t.Context(), release))
	delay, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.Positive(t, delay, "old status.stopped must not bypass readiness")
	meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type: v3beta1.ConditionHealthy, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonSucceeded,
		Message: "all workloads are ready", ObservedGeneration: dogu.Generation,
	})
	require.NoError(t, k8s.Status().Update(t.Context(), dogu))
	delay, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	assert.Zero(t, delay)
	assert.False(t, dogu.Status.Stopped)
	assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionReady))

	// Stop again with unchanged input.
	dogu.Spec.Stopped = true
	dogu.Generation++
	require.NoError(t, k8s.Update(t.Context(), dogu))
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(release), release))
	release.Spec.Suspend = true
	release.Generation++
	require.NoError(t, k8s.Update(t.Context(), release))
	release.Status.ObservedGeneration = release.Generation
	require.NoError(t, k8s.Status().Update(t.Context(), release))
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	require.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))

	// Repeated metadata failures preserve their diagnostic without rewriting the status.
	chart.metaError = assert.AnError
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	pendingFailure := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	require.NotNil(t, pendingFailure)
	assert.Equal(t, metav1.ConditionUnknown, pendingFailure.Status)
	assert.Contains(t, pendingFailure.Message, assert.AnError.Error())
	resourceVersion := dogu.ResourceVersion
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.ErrorIs(t, err, assert.AnError)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	assert.Equal(t, resourceVersion, dogu.ResourceVersion)
	chart.metaError = nil
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	require.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	generation := dogu.Generation

	// Global input changes without a new Dogu generation. A source failure
	// aborts before comparing that input, so the previous False is now unknown.
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(global), global))
	global.Data["config.yaml"] = "fqdn: changed.example"
	require.NoError(t, k8s.Update(t.Context(), global))
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(repository), repository))
	repository.Status.Conditions = []metav1.Condition{
		{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionFalse},
		{Type: source.FetchFailedCondition, Status: metav1.ConditionTrue, Reason: source.OCIPullFailedReason, Message: "chart not found"},
	}
	require.NoError(t, k8s.Update(t.Context(), repository))
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	assert.Equal(t, generation, dogu.Generation)
	pending := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	require.NotNil(t, pending)
	assert.Equal(t, metav1.ConditionUnknown, pending.Status)
	assert.True(t, dogu.Status.Stopped)

	// Restore unchanged input. A subsequent schema abort must not invalidate
	// the comparison that has already completed during this pass.
	global.Data["config.yaml"] = "fqdn: ces.example"
	require.NoError(t, k8s.Update(t.Context(), global))
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(repository), repository))
	repository.Status.Conditions = []metav1.Condition{{Type: fluxmeta.ReadyCondition, Status: metav1.ConditionTrue}}
	require.NoError(t, k8s.Update(t.Context(), repository))
	chart.validationError = assert.AnError
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	assert.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
	assert.True(t, meta.IsStatusConditionFalse(dogu.Status.Conditions, v3beta1.ConditionValid))

	// Pausing explicitly skips reconciliation rather than attempting a comparison.
	pending = meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending).DeepCopy()
	dogu.Spec.PauseReconciliation = true
	dogu.Generation++
	require.NoError(t, k8s.Update(t.Context(), dogu))
	_, err = useCase.HandleUntilApplied(t.Context(), dogu)
	require.NoError(t, err)
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	assert.Equal(t, pending, meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending))
}
