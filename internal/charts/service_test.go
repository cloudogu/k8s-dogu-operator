package charts

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helmChart "helm.sh/helm/v3/pkg/chart"
	chartLoaderPkg "helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// staticLoader builds a mockChartLoader that always returns the given digest and chart archive with
// no errors. Tests that need errors or a changing sequence of returns configure the mock inline.
func staticLoader(digest string, archive []byte) *mockChartLoader {
	return &mockChartLoader{
		ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
			return digest, nil
		},
		GetChartArchiveFunc: func(context.Context, *v3beta1.Dogu) ([]byte, error) {
			return archive, nil
		},
	}
}

func testDogu(namespace, name string) *v3beta1.Dogu {
	return &v3beta1.Dogu{ObjectMeta: metaV1.ObjectMeta{Namespace: namespace, Name: name}}
}

// stubResolver is a releaseResolver that returns a fixed operation or error.
type stubResolver struct {
	op  flux.ReleaseOperation
	err error
}

func (s stubResolver) ResolveOperation(context.Context, string, string) (flux.ReleaseOperation, error) {
	return s.op, s.err
}

// installResolver resolves every dogu to a first-install operation; used by tests that don't
// exercise the install/upgrade distinction.
func installResolver() stubResolver {
	return stubResolver{op: flux.ReleaseOperation{IsInstall: true, Revision: 1}}
}

// testChartArchive builds a valid, minimal Helm chart archive (optionally carrying extra Files) and
// returns its raw .tgz bytes, mirroring what the ChartProvider downloads.
func testChartArchive(t *testing.T, name string, files ...*helmChart.File) []byte {
	t.Helper()
	c := &helmChart.Chart{
		Metadata: &helmChart.Metadata{APIVersion: helmChart.APIVersionV2, Name: name, Version: "0.1.0"},
		Files:    files,
	}
	return saveChartArchive(t, c)
}

// chartDirArchive loads a chart from a testdata directory and returns it as .tgz bytes.
func chartDirArchive(t *testing.T, dir string) []byte {
	t.Helper()
	raw, err := chartLoaderPkg.LoadDir(dir)
	require.NoError(t, err)
	return saveChartArchive(t, raw)
}

func saveChartArchive(t *testing.T, c *helmChart.Chart) []byte {
	t.Helper()
	dir := t.TempDir()
	path, err := chartutil.Save(c, dir)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func Test_Service_chartFor(t *testing.T) {
	ctx := context.Background()
	caps := chartutil.DefaultCapabilities

	t.Run("loads and caches on first call", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", testChartArchive(t, "cas"))
		s := NewService(loader, nil, caps, installResolver())

		c, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.NoError(t, err)
		assert.Equal(t, "cas", c.raw.Name())
		// ReleaseRef is derived by the Service from the dogu, not returned by the loader.
		assert.Equal(t, ReleaseRef{Name: "cas", Namespace: "ecosystem"}, c.ref)
		assert.Same(t, caps, c.caps)
		assert.Equal(t, 1, len(loader.GetChartArchiveCalls()))
		assert.Equal(t, 1, len(loader.ArtifactDigestCalls()))
	})

	t.Run("reuses cached bytes but parses a fresh chart when digest is unchanged", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", testChartArchive(t, "cas"))
		s := NewService(loader, nil, caps, installResolver())
		dogu := testDogu("ecosystem", "cas")

		first, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)
		second, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		assert.NotSame(t, first, second, "each call must parse an independently mutable chart")
		assert.NotSame(t, first.raw, second.raw, "the parsed chart must not be shared across calls")
		assert.Equal(t, first.raw.Name(), second.raw.Name())
		assert.Equal(t, 1, len(loader.GetChartArchiveCalls()), "download must not run again on a cache hit")
		assert.Equal(t, 2, len(loader.ArtifactDigestCalls()), "digest is checked every call")
	})

	t.Run("reloads when digest changes", func(t *testing.T) {
		digests := []string{"sha256:aaa", "sha256:bbb"}
		archives := [][]byte{testChartArchive(t, "cas"), testChartArchive(t, "cas-v2")}
		var digestIdx, archiveIdx int
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				d := digests[digestIdx]
				digestIdx++
				return d, nil
			},
			GetChartArchiveFunc: func(context.Context, *v3beta1.Dogu) ([]byte, error) {
				a := archives[archiveIdx]
				archiveIdx++
				return a, nil
			},
		}
		s := NewService(loader, nil, caps, installResolver())
		dogu := testDogu("ecosystem", "cas")

		first, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		second, err := s.chartFor(ctx, dogu)
		require.NoError(t, err)

		assert.NotSame(t, first, second)
		assert.Equal(t, "cas-v2", second.raw.Name())
		assert.Equal(t, 2, len(loader.GetChartArchiveCalls()))
	})

	t.Run("caches per dogu", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", testChartArchive(t, "x"))
		s := NewService(loader, nil, caps, installResolver())

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))
		require.NoError(t, err)
		_, err = s.chartFor(ctx, testDogu("ecosystem", "nginx"))
		require.NoError(t, err)

		assert.Equal(t, 2, len(loader.GetChartArchiveCalls()), "different dogus must be cached separately")
	})

	t.Run("propagates digest error without loading", func(t *testing.T) {
		// GetChartArchiveFunc is intentionally unset: GetChartArchive must never be reached.
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, caps, installResolver())

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
		assert.Equal(t, 0, len(loader.GetChartArchiveCalls()))
	})

	t.Run("propagates load error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "sha256:aaa", nil
			},
			GetChartArchiveFunc: func(context.Context, *v3beta1.Dogu) ([]byte, error) {
				return nil, errors.New("boom")
			},
		}
		s := NewService(loader, nil, caps, installResolver())

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart archive")
	})

	t.Run("fails when cached bytes are not a valid chart archive", func(t *testing.T) {
		loader := staticLoader("sha256:aaa", []byte("not a chart archive"))
		s := NewService(loader, nil, caps, installResolver())

		_, err := s.chartFor(ctx, testDogu("ecosystem", "cas"))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse chart archive")
	})
}

func Test_Service_Evict(t *testing.T) {
	ctx := context.Background()
	loader := staticLoader("sha256:aaa", testChartArchive(t, "cas"))
	s := NewService(loader, nil, chartutil.DefaultCapabilities, installResolver())
	dogu := testDogu("ecosystem", "cas")

	_, err := s.chartFor(ctx, dogu)
	require.NoError(t, err)

	s.Evict(dogu)

	_, err = s.chartFor(ctx, dogu)
	require.NoError(t, err)
	assert.Equal(t, 2, len(loader.GetChartArchiveCalls()), "eviction must force a re-download on the next access")
}

func Test_Service_ChartPatchTemplate(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	t.Run("returns patch template file when present", func(t *testing.T) {
		archive := testChartArchive(t, "cas",
			&helmChart.File{Name: "extra.yaml", Data: []byte("nope")},
			&helmChart.File{Name: patchTemplateFileName, Data: []byte("patch: data")},
		)
		s := NewService(staticLoader("d", archive), nil, chartutil.DefaultCapabilities, installResolver())

		data, ok, err := s.ChartPatchTemplate(ctx, dogu)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("patch: data"), data)
	})

	t.Run("reports absent when no patch template file", func(t *testing.T) {
		archive := testChartArchive(t, "cas", &helmChart.File{Name: "extra.yaml", Data: []byte("x: 1")})
		s := NewService(staticLoader("d", archive), nil, chartutil.DefaultCapabilities, installResolver())

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
		s := NewService(loader, nil, chartutil.DefaultCapabilities, installResolver())

		_, _, err := s.ChartPatchTemplate(ctx, dogu)

		require.Error(t, err)
	})
}

func Test_Service_DoguMetaValues(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	t.Run("returns dogu meta values file when present", func(t *testing.T) {
		archive := testChartArchive(t, "cas",
			&helmChart.File{Name: "extra.yaml", Data: []byte("nope")},
			&helmChart.File{Name: doguValuesMetaFileName, Data: []byte("meta: data")},
		)
		s := NewService(staticLoader("d", archive), nil, chartutil.DefaultCapabilities, installResolver())

		data, ok, err := s.DoguMetaValues(ctx, dogu)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("meta: data"), data)
	})

	t.Run("reports absent when no dogu meta values file", func(t *testing.T) {
		archive := testChartArchive(t, "cas", &helmChart.File{Name: "extra.yaml", Data: []byte("x: 1")})
		s := NewService(staticLoader("d", archive), nil, chartutil.DefaultCapabilities, installResolver())

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
		s := NewService(staticLoader("d", chartDirArchive(t, "testdata/testchart")), nil, chartutil.DefaultCapabilities, installResolver())

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
		s := NewService(loader, nil, chartutil.DefaultCapabilities, installResolver())

		_, err := s.Render(ctx, dogu, map[string]any{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
	})

	t.Run("propagates release-operation resolver error", func(t *testing.T) {
		s := NewService(staticLoader("d", chartDirArchive(t, "testdata/testchart")), nil, chartutil.DefaultCapabilities,
			stubResolver{err: errors.New("boom")})

		_, err := s.Render(ctx, dogu, map[string]any{"image": "nginx:2.0.0"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to resolve release operation")
	})
}

func Test_Service_ValidateValues(t *testing.T) {
	ctx := context.Background()
	dogu := testDogu("ecosystem", "cas")

	// A schema violation proves ValidateValues actually reaches the chart's schema through the
	// Service; the schema cases themselves are owned by Test_chart_validateValues.
	t.Run("rejects values that violate the chart schema", func(t *testing.T) {
		s := NewService(staticLoader("d", chartDirArchive(t, "testdata/testchart")), nil, chartutil.DefaultCapabilities, installResolver())

		err := s.ValidateValues(ctx, dogu, map[string]any{"replicaCount": "three"})

		require.Error(t, err)
		assert.ErrorContains(t, err, "schema validation")
	})

	t.Run("propagates chartFor error", func(t *testing.T) {
		loader := &mockChartLoader{
			ArtifactDigestFunc: func(context.Context, *v3beta1.Dogu) (string, error) {
				return "", errors.New("boom")
			},
		}
		s := NewService(loader, nil, chartutil.DefaultCapabilities, installResolver())

		err := s.ValidateValues(ctx, dogu, map[string]any{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get chart artifact digest")
	})
}
