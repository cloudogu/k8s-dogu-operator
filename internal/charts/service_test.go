package charts

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helmChart "helm.sh/helm/v3/pkg/chart"
	chartLoaderPkg "helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// staticLoader builds a mockChartLoader that always returns the given digest and raw chart with no
// errors. Tests that need errors or a changing sequence of returns configure the mock inline.
func staticLoader(digest string, raw *helmChart.Chart) *mockChartLoader {
	return &mockChartLoader{
		ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
			return digest, nil
		},
		GetChartFunc: func(context.Context, *v3beta1.Dogu) (*helmChart.Chart, error) {
			return raw, nil
		},
	}
}

func testDogu(namespace, name string) *v3beta1.Dogu {
	return &v3beta1.Dogu{ObjectMeta: metaV1.ObjectMeta{Namespace: namespace, Name: name}}
}

func testRawChart(name string) *helmChart.Chart {
	return &helmChart.Chart{Metadata: &helmChart.Metadata{Name: name}}
}

func Test_Service_chartFor(t *testing.T) {
	ctx := context.Background()
	caps := chartutil.DefaultCapabilities

	t.Run("loads and caches on first call", func(t *testing.T) {
		raw := testRawChart("cas")
		loader := staticLoader("sha256:aaa", raw)
		s := NewService(loader, nil, caps)

		c, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.NoError(t, err)
		assert.Same(t, raw, c.raw)
		// ReleaseRef is derived by the Service from the dogu, not returned by the loader.
		assert.Equal(t, ReleaseRef{Name: "cas", Namespace: "ecosystem"}, c.ref)
		assert.Same(t, caps, c.caps)
		assert.Equal(t, "sha256:aaa", c.digest)
		assert.Equal(t, 1, len(loader.GetChartCalls()))
		assert.Equal(t, 1, len(loader.ArtifactDigestCalls()))
	})

	t.Run("reuses cached chart when digest is unchanged", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", testRawChart("cas"))
		s := NewService(loader, nil, caps)
		dogu := testDogu("ecosystem", "cas")

		first, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)
		second, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		assert.Same(t, first, second, "same digest must return the same cached instance")
		assert.Equal(t, 1, len(loader.GetChartCalls()), "load must not run again on a cache hit")
		assert.Equal(t, 2, len(loader.ArtifactDigestCalls()), "digest is checked every call")
	})

	t.Run("reloads when digest changes", func(t *testing.T) {
		digests := []string{"sha256:aaa", "sha256:bbb"}
		charts := []*helmChart.Chart{testRawChart("cas"), testRawChart("cas-v2")}
		var digestIdx, chartIdx int
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				d := digests[digestIdx]
				digestIdx++
				return d, nil
			},
			GetChartFunc: func(context.Context, *v3beta1.Dogu) (*helmChart.Chart, error) {
				c := charts[chartIdx]
				chartIdx++
				return c, nil
			},
		}
		s := NewService(loader, nil, caps)
		dogu := testDogu("ecosystem", "cas")

		first, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		second, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		assert.NotSame(t, first, second)
		assert.Equal(t, "sha256:bbb", second.digest)
		assert.Equal(t, 2, len(loader.GetChartCalls()))
	})

	t.Run("caches per dogu", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", testRawChart("x"))
		s := NewService(loader, nil, caps)

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))
		require.NoError(t, err)
		_, err = s.chartFor(ctx, testDogu("ecosystem", "nginx"))
		require.NoError(t, err)

		assert.Equal(t, 2, len(loader.GetChartCalls()), "different dogus must be cached separately")
	})

	t.Run("propagates digest error without loading", func(t *testing.T) {
		// GetChartFunc is intentionally unset: GetChart must never be reached.
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, caps)

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
		assert.Equal(t, 0, len(loader.GetChartCalls()))
	})

	t.Run("propagates load error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "sha256:aaa", nil
			},
			GetChartFunc: func(context.Context, *v3beta1.Dogu) (*helmChart.Chart, error) {
				return nil, errors.New("boom")
			},
		}
		s := NewService(loader, nil, caps)

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to load chart")
	})
}

func Test_Service_Evict(t *testing.T) {
	ctx := context.Background()
	loader := staticLoader("sha256:aaa", testRawChart("cas"))
	s := NewService(loader, nil, chartutil.DefaultCapabilities)
	dogu := testDogu("ecosystem", "cas")

	_, err := s.chartFor(ctx, dogu)
	require.NoError(t, err)

	s.Evict(dogu)

	_, err = s.chartFor(ctx, dogu)
	require.NoError(t, err)
	assert.Equal(t, 2, len(loader.GetChartCalls()), "eviction must force a reload on the next access")
}

func Test_Service_ChartPatchTemplate(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	t.Run("returns patch template file when present", func(t *testing.T) {
		raw := testRawChart("cas")
		raw.Files = []*helmChart.File{
			{Name: "extra.yaml", Data: []byte("nope")},
			{Name: patchTemplateFileName, Data: []byte("patch: data")},
		}
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		data, ok, err := s.ChartPatchTemplate(ctx, dogu)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("patch: data"), data)
	})

	t.Run("reports absent when no patch template file", func(t *testing.T) {
		raw := testRawChart("cas")
		raw.Files = []*helmChart.File{{Name: "values.yaml", Data: []byte("x: 1")}}
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		data, ok, err := s.ChartPatchTemplate(ctx, dogu)

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, data)
	})

	t.Run("propagates loader error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, chartutil.DefaultCapabilities)

		_, _, err := s.ChartPatchTemplate(ctx, dogu)

		require.Error(t, err)
	})
}

func Test_Service_DoguMetaValues(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	t.Run("returns dogu meta values file when present", func(t *testing.T) {
		raw := testRawChart("cas")
		raw.Files = []*helmChart.File{
			{Name: "extra.yaml", Data: []byte("nope")},
			{Name: doguValuesMetaFileName, Data: []byte("meta: data")},
		}
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		data, ok, err := s.DoguMetaValues(ctx, dogu)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("meta: data"), data)
	})

	t.Run("reports absent when no dogu meta values file", func(t *testing.T) {
		raw := testRawChart("cas")
		raw.Files = []*helmChart.File{{Name: "values.yaml", Data: []byte("x: 1")}}
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		_, ok, err := s.DoguMetaValues(ctx, dogu)

		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func Test_Service_Render(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	// Render delegation and restConfig pass-through. The fixture chart is lookup-free, so a nil
	// restConfig renders fine; the deep render semantics are owned by Test_chart_render.
	t.Run("renders the dogu chart through the cached chart", func(t *testing.T) {
		raw, err := chartLoaderPkg.LoadDir("testdata/testchart")
		require.NoError(t, err)
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		objs, err := s.Render(ctx, dogu, map[string]any{"image": "nginx:2.0.0"})

		require.NoError(t, err)
		assert.NotEmpty(t, objs)
	})

	t.Run("propagates chartFor error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, chartutil.DefaultCapabilities)

		_, err := s.Render(ctx, dogu, map[string]any{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
	})
}

func Test_Service_ValidateValues(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	// A schema violation proves ValidateValues actually reaches the chart's schema through the
	// Service; the schema cases themselves are owned by Test_chart_validateValues.
	t.Run("rejects values that violate the chart schema", func(t *testing.T) {
		raw, err := chartLoaderPkg.LoadDir("testdata/testchart")
		require.NoError(t, err)
		s := NewService(staticLoader("d", raw), nil, chartutil.DefaultCapabilities)

		err = s.ValidateValues(ctx, dogu, map[string]any{"replicaCount": "three"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "schema validation")
	})

	t.Run("propagates chartFor error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, chartutil.DefaultCapabilities)

		err := s.ValidateValues(ctx, dogu, map[string]any{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
	})
}
