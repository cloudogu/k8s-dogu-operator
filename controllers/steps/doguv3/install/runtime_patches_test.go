package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/charts"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	helmflux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type runtimeChartArchive []byte

func (a runtimeChartArchive) ArtifactDigest(context.Context, *v3beta1.Dogu) (string, error) {
	return "fixture-digest", nil
}

func (a runtimeChartArchive) GetChartArchive(context.Context, *v3beta1.Dogu) ([]byte, error) {
	return a, nil
}

// Exercise the real archive/service/assembler/validator path, without a cluster or Flux installation.
func TestRuntimePatchesHelmRelease(t *testing.T) {
	ctx := t.Context()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	require.NoError(t, helmflux.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "fixture", Namespace: "ecosystem", UID: "fixture-uid"},
		Spec: v3beta1.DoguSpec{
			Name: "fixture", Version: "1.0.0", DoguApiVersion: "v3",
			Values:       apiext.JSON{Raw: []byte(`{"explicit":"user"}`)},
			MappedValues: map[string]string{"setting": "selected"},
		},
	}
	global := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "global-config", Namespace: dogu.Namespace},
		Data:       map[string]string{"config.yaml": "fqdn: first.example\n"},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogu, global).WithStatusSubresource(dogu).Build()
	chart := &helmchart.Chart{
		Metadata: &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: "fixture", Version: "1.0.0"},
		Values:   map[string]any{"host": "chart-default", "untouched": "chart-only"},
		Raw:      []*helmchart.File{{Name: "values.yaml", Data: []byte("host: chart-default\nuntouched: chart-only\n")}},
		Schema:   []byte(`{"type":"object","required":["host"],"properties":{"host":{"type":"string","minLength":1},"integer":{"enum":[9007199254740993]}}}`),
		Files: []*helmchart.File{
			{Name: "chart-patch-tpl.yaml", Data: []byte(`---
apiVersion: v1
patches:
  values.yaml:
    image: {{ registryFrom .images.fixture }}
runtimePatches:
  values.yaml:
    host: {{ globalConfig "fqdn" | default "" | quote }}
    explicit: runtime
    mapped: runtime
    integer: 9007199254740993
    backend:
      applicationHost: {{ globalConfig "fqdn" | default "" | quote }}
`)},
			{Name: "dogu-values-metadata.yaml", Data: []byte(`apiVersion: v1
metavalues:
  setting:
    keys:
      - path: mapped
        mapping:
          selected: mapped-wins
`)},
		},
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: rendered
data:
  host: {{ .Values.host | quote }}
  untouched: {{ .Values.untouched | quote }}
  integer: {{ .Values.integer | quote }}
`)}},
	}
	chart.AddDependency(&helmchart.Chart{
		Metadata: &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: "backend", Version: "1.0.0"},
		Raw:      []*helmchart.File{{Name: "values.yaml", Data: []byte("applicationHost: upstream-default\n")}},
		Values:   map[string]any{"applicationHost": "upstream-default"},
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: backend
data:
  applicationHost: {{ .Values.applicationHost | quote }}
`)}},
	})
	path, err := chartutil.Save(chart, t.TempDir())
	require.NoError(t, err)
	archive, err := os.ReadFile(path)
	require.NoError(t, err)
	service := charts.NewService(runtimeChartArchive(archive), &rest.Config{}, chartutil.DefaultCapabilities, flux.NewHelmReleaseReader(cl))
	step := NewEnsureHelmReleaseStep(cl, &config.OperatorConfig{
		DoguHelmRetryInterval: time.Minute, DoguHelmReconciliationInterval: time.Minute,
	}, service, record.NewFakeRecorder(10))
	validateStep := NewValidateChartStep(service, values.NewAssembler(cl), cl)

	for _, host := range []string{"first.example", "changed.example", ""} {
		require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(global), global))
		global.Data["config.yaml"] = "fqdn: " + host + "\n"
		require.NoError(t, cl.Update(ctx, global))
		require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(dogu), dogu))
		result := validateStep.Run(ctx, dogu)
		if result.Continue {
			result = step.Run(ctx, dogu)
		}
		release := &helmflux.HelmRelease{}
		require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(dogu), release))
		var persisted map[string]any
		decoder := json.NewDecoder(bytes.NewReader(release.Spec.Values.Raw))
		decoder.UseNumber()
		require.NoError(t, decoder.Decode(&persisted))
		if host == "" {
			assert.False(t, result.Continue)
			assertCondition(t, dogu, v3beta1.ConditionValid, metav1.ConditionFalse, v3beta1.ReasonSchemaInvalid)
			assert.Equal(t, "changed.example", persisted["host"], "invalid updates leave the existing release untouched")
			continue
		}
		require.True(t, result.Continue, "%+v", result)
		assert.Equal(t, host, persisted["host"])
		assert.Equal(t, "user", persisted["explicit"])
		assert.Equal(t, "mapped-wins", persisted["mapped"])
		assert.Equal(t, json.Number("9007199254740993"), persisted["integer"])
		assert.NotContains(t, persisted, "untouched", "Helm chart defaults must not be persisted as overrides")
		objects, err := service.Render(ctx, dogu, persisted)
		require.NoError(t, err)
		require.Len(t, objects, 2)
		for _, object := range objects {
			if object.GetName() == "backend" {
				assert.Equal(t, map[string]any{"applicationHost": host}, object.Object["data"])
			} else {
				assert.Equal(t, "rendered", object.GetName())
				assert.Equal(t, map[string]any{
					"host": host, "untouched": "chart-only", "integer": "9007199254740993",
				}, object.Object["data"])
			}
		}
	}
}

func TestRuntimePatchesValidationFailures(t *testing.T) {
	for _, tt := range []struct {
		name, template, errorContains string
		renderFailure                 bool
	}{
		{name: "template", errorContains: "mandatory value missing", template: `apiVersion: v1
runtimePatches:
  values.yaml:
    host: {{ fail "mandatory value missing" }}
`},
		{name: "render", renderFailure: true},
	} {
		for _, skip := range []bool{false, true} {
			t.Run(tt.name+"/skipSchema="+fmt.Sprint(skip), func(t *testing.T) {
				cl, dogu := runtimeFailureFixture(t, "fqdn: new.example")
				dogu.Spec.SkipSchemaValidation = skip
				service := NewMockChartService(t)
				service.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil).Once()
				service.EXPECT().ChartPatchTemplate(mock.Anything, dogu).Return([]byte(tt.template), tt.template != "", nil).Once()
				if tt.renderFailure {
					if !skip {
						service.EXPECT().ValidateValues(mock.Anything, dogu, mock.Anything).Return(nil).Once()
					}
					service.EXPECT().Render(mock.Anything, dogu, mock.Anything).Return(nil, assert.AnError).Once()
				}
				step := NewValidateChartStep(service, values.NewAssembler(cl), cl)
				result := step.Run(t.Context(), dogu)
				assert.False(t, result.Continue)
				if tt.renderFailure {
					assertCondition(t, dogu, v3beta1.ConditionValid, metav1.ConditionFalse, ReasonRenderFailed)
				} else {
					require.ErrorContains(t, result.Err, tt.errorContains)
				}
			})
		}
	}
}

func TestRuntimePatchesWriteFailuresPreserveHelmRelease(t *testing.T) {
	for _, tt := range []struct {
		name, config, template, errorContains string
	}{
		{name: "template", config: "fqdn: new.example", errorContains: "mandatory value missing", template: `apiVersion: v1
runtimePatches:
  values.yaml:
    host: {{ fail "mandatory value missing" }}
`},
		{name: "serialization", config: "invalid: .nan", errorContains: "unsupported value: NaN"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cl, dogu := runtimeFailureFixture(t, tt.config)
			release := &helmflux.HelmRelease{ObjectMeta: metav1.ObjectMeta{Name: dogu.Name, Namespace: dogu.Namespace},
				Spec: helmflux.HelmReleaseSpec{
					ReleaseName: "last-valid-release", Suspend: true, Interval: metav1.Duration{Duration: 7 * time.Minute},
					ChartRef: &helmflux.CrossNamespaceSourceReference{Kind: "OCIRepository", Name: "last-valid-chart"},
					Values:   &apiext.JSON{Raw: []byte(`{"host":"last-valid.example"}`)},
				}}
			require.NoError(t, cl.Create(t.Context(), release))
			require.NoError(t, cl.Get(t.Context(), client.ObjectKeyFromObject(release), release))
			before := release.DeepCopy()
			service := NewMockChartService(t)
			service.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil).Once()
			service.EXPECT().ChartPatchTemplate(mock.Anything, dogu).Return([]byte(tt.template), tt.template != "", nil).Once()
			step := NewEnsureHelmReleaseStep(cl, &config.OperatorConfig{}, service, record.NewFakeRecorder(10))
			result := step.Run(t.Context(), dogu)
			assert.False(t, result.Continue)
			require.ErrorContains(t, result.Err, tt.errorContains)
			require.NoError(t, cl.Get(t.Context(), client.ObjectKeyFromObject(release), release))
			assert.Equal(t, before, release, "failure must not change any part of the last valid release")
		})
	}
}

func runtimeFailureFixture(t *testing.T, configYAML string) (client.Client, *v3beta1.Dogu) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	require.NoError(t, helmflux.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "fixture", Namespace: "ecosystem"},
		Spec: v3beta1.DoguSpec{Name: "fixture"}}
	global := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "global-config", Namespace: dogu.Namespace},
		Data: map[string]string{"config.yaml": configYAML}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogu, global).WithStatusSubresource(dogu).Build()
	return cl, dogu
}

func TestHelmNumberFidelity(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, helmflux.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	dogu := &v3beta1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "fixture", Namespace: "ecosystem"}, Spec: v3beta1.DoguSpec{Name: "fixture"}}
	chart := &helmchart.Chart{
		Metadata: &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: "fixture", Version: "1.0.0"},
		Schema:   []byte(`{"type":"object","properties":{"integer":{"enum":[9007199254740993,-9223372036854775808,18446744073709551615]}}}`),
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: rendered
data:
  integer: {{ .Values.integer | quote }}
`)}},
	}
	path, err := chartutil.Save(chart, t.TempDir())
	require.NoError(t, err)
	archive, err := os.ReadFile(path)
	require.NoError(t, err)
	service := charts.NewService(runtimeChartArchive(archive), &rest.Config{}, chartutil.DefaultCapabilities, flux.NewHelmReleaseReader(cl))
	// Pass values directly: the fake Kubernetes client's storage round-trips unsigned integers through float64.
	for _, integer := range []string{"9007199254740993", "-9223372036854775808", "18446744073709551615"} {
		values := map[string]any{"integer": json.Number(integer)}
		require.NoError(t, service.ValidateValues(t.Context(), dogu, values))
		objects, renderErr := service.Render(t.Context(), dogu, values)
		require.NoError(t, renderErr)
		require.Len(t, objects, 1)
		assert.Equal(t, map[string]any{"integer": integer}, objects[0].Object["data"])
	}
	assert.Error(t, service.ValidateValues(t.Context(), dogu, map[string]any{"integer": json.Number("9007199254740992")}))
}
