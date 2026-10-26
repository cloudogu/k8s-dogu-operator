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
	"gopkg.in/yaml.v3"
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

const (
	runtimeTestDoguName  = "fixture"
	runtimeTestNamespace = "ecosystem"
)

type runtimeChartArchive []byte

func (a runtimeChartArchive) ArtifactDigest(context.Context, *v3beta1.Dogu) (string, error) {
	return "fixture-digest", nil
}

func (a runtimeChartArchive) GetChartArchive(context.Context, *v3beta1.Dogu) ([]byte, error) {
	return a, nil
}

// TestRuntimePatchesHelmRelease exercises the real archive/service/assembler/validator path,
// without a cluster or Flux installation.
func TestRuntimePatchesHelmRelease(t *testing.T) {
	ctx := t.Context()
	mirrorPatchTemplate := `---
apiVersion: v1
patches:
  values.yaml:
    image: '{{ registryFrom .images.fixture }}'
`
	runtimePatchTemplate := `host: '{{ globalConfig "fqdn" }}'
explicit: runtime
mapped: runtime
integer: 9007199254740993
backend:
  applicationHost: '{{ globalConfig "fqdn" }}'
`
	valuesMetadata := `apiVersion: v1
metavalues:
  setting:
    keys:
      - path: mapped
        mapping:
          selected: mapped-wins
`
	parentTemplate := `apiVersion: v1
kind: ConfigMap
metadata:
  name: rendered
data:
  host: {{ .Values.host | quote }}
  untouched: {{ .Values.untouched | quote }}
  integer: {{ .Values.integer | quote }}
`
	subchartTemplate := `apiVersion: v1
kind: ConfigMap
metadata:
  name: backend
data:
  applicationHost: {{ .Values.applicationHost | quote }}
`

	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v3beta1.AddToScheme(scheme))
	require.NoError(t, helmflux.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{
		Name: runtimeTestDoguName, Namespace: runtimeTestNamespace, UID: "fixture-uid",
		Spec: v3beta1.DoguSpec{
			Name: runtimeTestDoguName, Version: "1.0.0", DoguApiVersion: "v3",
			Values:       apiext.JSON{Raw: []byte(`{"explicit":"user"}`)},
			MappedValues: map[string]string{"setting": "selected"},
		},
	}
	global := &corev1.ConfigMap{
		Name: "global-config", Namespace: dogu.Namespace,
		Data: map[string]string{"config.yaml": "fqdn: first.example\n"},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogu, global).WithStatusSubresource(dogu).Build()
	chart := &helmchart.Chart{
		Metadata: &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: runtimeTestDoguName, Version: "1.0.0"},
		Values:   map[string]any{"host": "chart-default", "untouched": "chart-only"},
		Raw:      []*helmchart.File{{Name: "values.yaml", Data: []byte("host: chart-default\nuntouched: chart-only\n")}},
		Schema:   []byte(`{"type":"object","required":["host"],"properties":{"host":{"type":"string","minLength":1},"integer":{"enum":[9007199254740993]}}}`),
		Files: []*helmchart.File{
			{Name: "mirror-patch-tpl.yaml", Data: []byte(mirrorPatchTemplate)},
			{Name: values.RuntimePatchTemplateFileName, Data: []byte(runtimePatchTemplate)},
			{Name: "dogu-values-metadata.yaml", Data: []byte(valuesMetadata)},
		},
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(parentTemplate)}},
	}
	chart.AddDependency(&helmchart.Chart{
		Metadata:  &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: "backend", Version: "1.0.0"},
		Raw:       []*helmchart.File{{Name: "values.yaml", Data: []byte("applicationHost: upstream-default\n")}},
		Values:    map[string]any{"applicationHost": "upstream-default"},
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(subchartTemplate)}},
	})
	archive := saveTestChartToArchive(t, chart)
	service := charts.NewService(archive, &rest.Config{}, chartutil.DefaultCapabilities, flux.NewHelmReleaseReader(cl))
	step := NewEnsureHelmReleaseStep(cl, &config.OperatorConfig{
		DoguHelmRetryInterval: time.Minute, DoguHelmReconciliationInterval: time.Minute,
	}, service, values.NewAssembler(cl), record.NewFakeRecorder(10))
	validateStep := NewValidateChartStep(service, values.NewAssembler(cl), cl)

	// The fixture checks string preservation, not hostname validity. The schema rejects only empty strings.
	for _, host := range []string{"first.example", "O'Reilly: #quoted\n", "changed.example", ""} {
		require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(global), global))
		configYAML, err := yaml.Marshal(map[string]string{"fqdn": host})
		require.NoError(t, err)
		global.Data["config.yaml"] = string(configYAML)
		require.NoError(t, cl.Update(ctx, global))
		require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(dogu), dogu))

		// Validate before writing, as the installation use case does.
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
		require.True(t, result.Continue, "result: %+v", result)
		assert.Equal(t, host, persisted["host"])
		assert.Equal(t, "user", persisted["explicit"])
		assert.Equal(t, "mapped-wins", persisted["mapped"])
		assert.Equal(t, json.Number("9007199254740993"), persisted["integer"])
		assert.NotContains(t, persisted, "untouched", "Helm chart defaults must not be persisted as overrides")

		// Render the persisted overrides to check consumption by both parent and subchart.
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
	const patch = `host: '{{ globalConfig "absent" }}'
`
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("skipSchema=%t", skip), func(t *testing.T) {
			cl, dogu := runtimeFailureFixture(t, "fqdn: new.example")
			dogu.Spec.SkipSchemaValidation = skip
			service := newMockChartService(t)
			service.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil).Once()
			service.EXPECT().RuntimePatchTemplate(mock.Anything, dogu).Return([]byte(patch), true, nil).Once()
			step := NewValidateChartStep(service, values.NewAssembler(cl), cl)

			result := step.Run(t.Context(), dogu)

			assert.False(t, result.Continue)
			require.ErrorContains(t, result.Err, `globalConfig "absent" is missing`)
		})
	}
}

func TestRuntimePatchesWriteFailuresPreserveHelmRelease(t *testing.T) {
	for _, tt := range []struct {
		name, config, template, errorContains string
	}{
		{
			name:          "missing config",
			config:        "fqdn: new.example",
			template:      `host: '{{ globalConfig "absent" }}'`,
			errorContains: `globalConfig "absent" is missing`,
		},
		{
			name:          "non-string config",
			config:        "fqdn: false",
			template:      `host: '{{ globalConfig "fqdn" }}'`,
			errorContains: "does not refer to a string",
		},
		{
			name:          "serialization",
			config:        "invalid: .nan",
			template:      "host: server.invalid",
			errorContains: "unsupported value: NaN",
		},
		{
			name:          "empty runtime file",
			config:        "fqdn: new.example",
			errorContains: "runtime-patch-tpl.yaml must not be empty",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			cl, dogu := runtimeFailureFixture(t, tt.config)
			release := &helmflux.HelmRelease{
				Name: dogu.Name, Namespace: dogu.Namespace,
				Spec: helmflux.HelmReleaseSpec{
					ReleaseName: "last-valid-release",
					Suspend:     true,
					Interval:    metav1.Duration{Duration: 7 * time.Minute},
					ChartRef:    &helmflux.CrossNamespaceSourceReference{Kind: "OCIRepository", Name: "last-valid-chart"},
					Values:      &apiext.JSON{Raw: []byte(`{"host":"last-valid.example"}`)},
				},
			}
			require.NoError(t, cl.Create(ctx, release))
			require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(release), release))
			before := release.DeepCopy()
			service := newMockChartService(t)
			service.EXPECT().DoguMetaValues(mock.Anything, dogu).Return(nil, false, nil).Once()
			service.EXPECT().RuntimePatchTemplate(mock.Anything, dogu).Return([]byte(tt.template), true, nil).Once()
			step := NewEnsureHelmReleaseStep(cl, &config.OperatorConfig{}, service, values.NewAssembler(cl), record.NewFakeRecorder(10))

			result := step.Run(ctx, dogu)

			assert.False(t, result.Continue)
			require.ErrorContains(t, result.Err, tt.errorContains)
			assert.ErrorContains(t, result.Err, dogu.Namespace+"/"+dogu.Name)
			require.NoError(t, cl.Get(ctx, client.ObjectKeyFromObject(release), release))
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
	dogu := &v3beta1.Dogu{
		Name: runtimeTestDoguName, Namespace: runtimeTestNamespace,
		Spec: v3beta1.DoguSpec{Name: runtimeTestDoguName},
	}
	global := &corev1.ConfigMap{
		Name: "global-config", Namespace: dogu.Namespace,
		Data: map[string]string{"config.yaml": configYAML},
	}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dogu, global).WithStatusSubresource(dogu).Build()
	return cl, dogu
}

func TestHelmNumberFidelity(t *testing.T) {
	template := `apiVersion: v1
kind: ConfigMap
metadata:
  name: rendered
data:
  integer: {{ .Values.integer | quote }}
`
	ctx := t.Context()
	scheme := runtime.NewScheme()
	require.NoError(t, helmflux.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	dogu := &v3beta1.Dogu{
		Name: runtimeTestDoguName, Namespace: runtimeTestNamespace,
		Spec: v3beta1.DoguSpec{Name: runtimeTestDoguName},
	}
	chart := &helmchart.Chart{
		Metadata:  &helmchart.Metadata{APIVersion: helmchart.APIVersionV2, Name: runtimeTestDoguName, Version: "1.0.0"},
		Schema:    []byte(`{"type":"object","properties":{"integer":{"enum":[9007199254740993,-9223372036854775808,18446744073709551615]}}}`),
		Templates: []*helmchart.File{{Name: "templates/configmap.yaml", Data: []byte(template)}},
	}
	archive := saveTestChartToArchive(t, chart)
	service := charts.NewService(archive, &rest.Config{}, chartutil.DefaultCapabilities, flux.NewHelmReleaseReader(cl))

	// Validate and render exact JSON numbers beyond float64's integer precision, including uint64's limit.
	for _, integer := range []string{"9007199254740993", "-9223372036854775808", "18446744073709551615"} {
		values := map[string]any{"integer": json.Number(integer)}
		require.NoError(t, service.ValidateValues(ctx, dogu, values))

		objects, renderErr := service.Render(ctx, dogu, values)

		require.NoError(t, renderErr)
		require.Len(t, objects, 1)
		assert.Equal(t, map[string]any{"integer": integer}, objects[0].Object["data"])
	}
	assert.Error(t, service.ValidateValues(ctx, dogu, map[string]any{"integer": json.Number("9007199254740992")}))
}

func saveTestChartToArchive(t *testing.T, chart *helmchart.Chart) runtimeChartArchive {
	t.Helper()
	path, err := chartutil.Save(chart, t.TempDir())
	require.NoError(t, err)
	archive, err := os.ReadFile(path)
	require.NoError(t, err)
	return runtimeChartArchive(archive)
}
