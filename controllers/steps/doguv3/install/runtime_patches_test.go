package install

import (
	"bytes"
	"context"
	"encoding/json"
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
			Values:       apiext.JSON{Raw: []byte(`{"explicit":"user","signed":-9223372036854775808}`)},
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
		Schema:   []byte(`{"type":"object","required":["host"],"properties":{"host":{"type":"string","minLength":1},"integer":{"enum":[9007199254740993]},"signed":{"enum":[-9223372036854775808]}}}`),
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
  signed: {{ .Values.signed | quote }}
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
		assert.Equal(t, json.Number("-9223372036854775808"), persisted["signed"])
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
					"signed": "-9223372036854775808",
				}, object.Object["data"])
			}
		}
	}
}
