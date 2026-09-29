package chart

import (
	"testing"

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

func Test_chart_render(t *testing.T) {
	c := loadTestChart(t)

	objs, err := c.render(map[string]any{"replicaCount": 3, "image": "nginx:2.0.0"}, nil)

	require.NoError(t, err)

	// Multi-doc split into two objects; _helpers.tpl and NOTES.txt excluded.
	nameToKind := map[string]string{}
	for _, o := range objs {
		u := o.(*unstructured.Unstructured)
		nameToKind[u.GetName()] = u.GetKind()
	}
	assert.Equal(t, map[string]string{
		"cas-web": "Deployment",
		"cas-svc": "Service",
	}, nameToKind)

	// .Values substitution + type fidelity: replicaCount rendered and decoded as int64.
	for _, o := range objs {
		u := o.(*unstructured.Unstructured)
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
	objs, err := c.render(map[string]any{}, nil)

	require.NoError(t, err)
	require.Len(t, objs, 2)
	for _, o := range objs {
		u := o.(*unstructured.Unstructured)
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
