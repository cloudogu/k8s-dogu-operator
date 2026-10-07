package charts

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// loadTestChart loads the lookup-free fixture chart from testdata. Because it uses no `lookup`,
// render works with a nil restConfig.
func loadTestChart(t *testing.T) *chart {
	t.Helper()
	raw, err := loader.LoadDir("testdata/testchart")
	require.NoError(t, err)
	return &chart{
		raw:  raw,
		ref:  ReleaseRef{Name: "cas", Namespace: "ecosystem"},
		caps: chartutil.DefaultCapabilities,
	}
}

// installOperation is the default release operation used by render tests that don't exercise the
// install/upgrade distinction themselves.
func installOperation() flux.ReleaseOperation {
	return flux.ReleaseOperation{IsInstall: true, Revision: 1}
}

func Test_chart_render_setsReleaseOperation(t *testing.T) {
	load := func(t *testing.T) *chart {
		t.Helper()
		raw, err := loader.LoadDir("testdata/releasechart")
		require.NoError(t, err)
		return &chart{raw: raw, ref: ReleaseRef{Name: "cas", Namespace: "ecosystem"}, caps: chartutil.DefaultCapabilities}
	}

	releaseInfo := func(t *testing.T, objs []*unstructured.Unstructured) map[string]string {
		t.Helper()
		for _, o := range objs {
			if o.GetName() == "release-info" {
				data, _, err := unstructured.NestedStringMap(o.Object, "data")
				require.NoError(t, err)
				return data
			}
		}
		t.Fatal("release-info ConfigMap not rendered")
		return nil
	}

	t.Run("install operation is rendered into .Release.*", func(t *testing.T) {
		objs, err := load(t).render(map[string]any{}, nil, flux.ReleaseOperation{IsInstall: true, Revision: 1})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"isInstall": "true", "isUpgrade": "false", "revision": "1"}, releaseInfo(t, objs))
	})

	t.Run("upgrade operation is rendered into .Release.*", func(t *testing.T) {
		objs, err := load(t).render(map[string]any{}, nil, flux.ReleaseOperation{IsUpgrade: true, Revision: 5})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"isInstall": "false", "isUpgrade": "true", "revision": "5"}, releaseInfo(t, objs))
	})
}

func Test_chart_render(t *testing.T) {
	c := loadTestChart(t)

	objs, err := c.render(map[string]any{"replicaCount": 3, "image": "nginx:2.0.0"}, nil, installOperation())

	require.NoError(t, err)

	// Multi-doc split into two objects; _helpers.tpl and NOTES.txt excluded.
	nameToKind := map[string]string{}
	for _, o := range objs {
		u := o
		nameToKind[u.GetName()] = u.GetKind()
	}
	assert.Equal(t, map[string]string{
		"cas-web": "Deployment",
		"cas-svc": "Service",
	}, nameToKind)

	// .Values substitution + type fidelity: replicaCount rendered and decoded as int64.
	for _, o := range objs {
		u := o
		if u.GetKind() == "Deployment" {
			replicas, found, err := unstructured.NestedInt64(u.Object, "spec", "replicas")
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, int64(3), replicas)
		}
	}
}

func Test_chart_render_usesDefaultsWhenValueOmitted(t *testing.T) {
	c := loadTestChart(t)

	// No overrides: values.yaml defaults must be coalesced in (replicaCount: 1).
	objs, err := c.render(map[string]any{}, nil, installOperation())

	require.NoError(t, err)
	require.Len(t, objs, 2)
	for _, o := range objs {
		u := o
		if u.GetKind() == "Deployment" {
			replicas, found, err := unstructured.NestedInt64(u.Object, "spec", "replicas")
			require.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, int64(1), replicas, "default replicaCount from values.yaml")
		}
	}
}

func Test_chart_validateValues(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]any
		wantErr string
	}{
		{
			name:   "valid values (coalesced with defaults) pass",
			values: map[string]any{"image": "nginx:1.2.3"},
		},
		{
			name:   "defaults alone satisfy the schema",
			values: map[string]any{},
		},
		{
			name:    "wrong type violates schema",
			values:  map[string]any{"replicaCount": "three"},
			wantErr: "schema validation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := loadTestChart(t)

			err := c.validateValues(tt.values)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// loadDepChart loads the fixture chart that declares a conditional `subchart` dependency.
func loadDepChart(t *testing.T) *chart {
	t.Helper()
	raw, err := loader.LoadDir("testdata/depchart")
	require.NoError(t, err)
	return &chart{
		raw:  raw,
		ref:  ReleaseRef{Name: "cas", Namespace: "ecosystem"},
		caps: chartutil.DefaultCapabilities,
	}
}

func Test_chart_render_processesDependencies(t *testing.T) {
	renderedNames := func(t *testing.T, enabled bool) map[string]struct{} {
		t.Helper()
		c := loadDepChart(t)

		objs, err := c.render(map[string]any{"subchart": map[string]any{"enabled": enabled}}, nil, installOperation())
		require.NoError(t, err)

		names := make(map[string]struct{}, len(objs))
		for _, o := range objs {
			names[o.GetName()] = struct{}{}
		}

		return names
	}

	t.Run("disabled subchart is pruned and not rendered", func(t *testing.T) {
		names := renderedNames(t, false)
		assert.Contains(t, names, "parent-cm")
		assert.NotContains(t, names, "subchart-cm", "disabled subchart must not be rendered")
	})

	t.Run("enabled subchart is rendered", func(t *testing.T) {
		names := renderedNames(t, true)
		assert.Contains(t, names, "parent-cm")
		assert.Contains(t, names, "subchart-cm")
	})
}

func Test_parseRenderedFilesToObjects(t *testing.T) {
	const cm = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n"

	tests := []struct {
		name      string
		files     map[string]string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single document",
			files:     map[string]string{"templates/a.yaml": cm},
			wantCount: 1,
		},
		{
			name: "multi-document split",
			files: map[string]string{
				"templates/a.yaml": "kind: ConfigMap\nmetadata:\n  name: a\n---\nkind: Secret\nmetadata:\n  name: b\n",
			},
			wantCount: 2,
		},
		{
			name:      "empty/whitespace content skipped",
			files:     map[string]string{"templates/a.yaml": "   \n\n"},
			wantCount: 0,
		},
		{
			name:      "empty docs between separators skipped",
			files:     map[string]string{"templates/a.yaml": "---\n---\n" + cm},
			wantCount: 1,
		},
		{
			name:      "NOTES.txt excluded by extension",
			files:     map[string]string{"templates/NOTES.txt": "hello there"},
			wantCount: 0,
		},
		{
			name:      "helpers tpl excluded by extension",
			files:     map[string]string{"templates/_helpers.tpl": "{{ define \"x\" }}"},
			wantCount: 0,
		},
		{
			name:      "non-yaml extension excluded",
			files:     map[string]string{"templates/x.json": `{"kind":"ConfigMap","metadata":{"name":"a"}}`},
			wantCount: 0,
		},
		{
			name:      "json content inside a yaml file is decoded",
			files:     map[string]string{"templates/a.yaml": `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"a"}}`},
			wantCount: 1,
		},
		{
			name:      "yml extension accepted",
			files:     map[string]string{"templates/a.yml": cm},
			wantCount: 1,
		},
		{
			name:    "invalid yaml returns error",
			files:   map[string]string{"templates/a.yaml": "key:\n\t- tab-indented"},
			wantErr: true,
		},
		{
			name:    "duplicate keys rejected",
			files:   map[string]string{"templates/a.yaml": "kind: ConfigMap\nmetadata:\n  labels:\n    app: a\n    app: b\n"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs, err := parseRenderedFilesToObjects(tt.files)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, objs, tt.wantCount)
			for _, o := range objs {
				assert.Implements(t, (*client.Object)(nil), o)
			}
		})
	}
}
