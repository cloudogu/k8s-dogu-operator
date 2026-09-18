package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHTTPChartArtifactLoader_Load(t *testing.T) {
	t.Run("loads a valid chart after verifying its digest", func(t *testing.T) {
		archive := createTestChartArchive(t)
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusOK, archive), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromBytes(archive).String())

		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Equal(t, "test-chart", actual.Name())
		assert.Equal(t, "bar", actual.Values["foo"])
		require.Len(t, actual.Templates, 1)
		assert.Equal(t, "templates/configmap.yaml", actual.Templates[0].Name)
	})

	t.Run("rejects an invalid digest before sending a request", func(t *testing.T) {
		client := NewMockHTTPClient(t)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", "not-a-digest")

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
		assert.ErrorContains(t, err, "failed to validate artifact digest")
	})

	t.Run("rejects an invalid URL before sending a request", func(t *testing.T) {
		client := NewMockHTTPClient(t)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "file:///tmp/chart.tgz", digest.SHA256.FromString("chart").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
		assert.ErrorContains(t, err, "failed to validate artifact URL")
	})

	t.Run("rejects a digest mismatch before parsing the archive", func(t *testing.T) {
		invalidArchive := []byte("not a chart")
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("https://source-controller/chart.tgz"))).Return(response(http.StatusOK, invalidArchive), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "https://source-controller/chart.tgz", digest.SHA256.FromString("different").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
		assert.ErrorContains(t, err, "digest mismatch")
		assert.NotContains(t, err.Error(), "failed to load chart artifact")
	})

	t.Run("rejects an invalid archive after successful verification", func(t *testing.T) {
		invalidArchive := []byte("not a chart")
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusOK, invalidArchive), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromBytes(invalidArchive).String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
		assert.ErrorContains(t, err, "failed to load chart artifact")
	})

	t.Run("rejects non-success status and closes the response body", func(t *testing.T) {
		body := &trackingReadCloser{Reader: strings.NewReader("missing")}
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(&http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: body}, nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromString("missing").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errChartArtifactNotFound)
		assert.ErrorContains(t, err, "unexpected HTTP status 404 Not Found")
		assert.True(t, body.closed)
	})

	t.Run("classifies server errors as temporary", func(t *testing.T) {
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusServiceUnavailable, nil), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromString("chart").String())

		assert.Nil(t, actual)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, errInvalidChartArtifact)
		assert.NotErrorIs(t, err, errChartArtifactNotFound)
	})

	t.Run("classifies client errors other than not found as invalid", func(t *testing.T) {
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusForbidden, nil), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromString("chart").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
	})

	t.Run("propagates download errors", func(t *testing.T) {
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(nil, assert.AnError)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromString("chart").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("rejects an artifact above the compressed size limit", func(t *testing.T) {
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusOK, make([]byte, maxChartArtifactSize+1)), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromString("irrelevant").String())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, errInvalidChartArtifact)
		assert.ErrorContains(t, err, "size exceeds")
	})

	t.Run("loads a chart with a packaged dependency", func(t *testing.T) {
		subchart := createChartArchive(t, map[string]string{
			"dependency/Chart.yaml": "apiVersion: v2\nname: dependency\nversion: 1.0.0\n",
		})
		archive := createChartArchive(t, map[string]string{
			"parent/Chart.yaml":            "apiVersion: v2\nname: parent\nversion: 1.0.0\n",
			"parent/charts/dependency.tgz": string(subchart),
		})
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusOK, archive), nil)

		actual, err := NewHTTPChartArtifactLoader(client).GetChart(testCtx, "http://source-controller/chart.tgz", digest.SHA256.FromBytes(archive).String())

		require.NoError(t, err)
		require.Len(t, actual.Dependencies(), 1)
		assert.Equal(t, "dependency", actual.Dependencies()[0].Name())
	})

	t.Run("serves a verified artifact from the cache on subsequent calls", func(t *testing.T) {
		archive := createTestChartArchive(t)
		expectedDigest := digest.SHA256.FromBytes(archive).String()
		client := NewMockHTTPClient(t)
		client.EXPECT().Do(mock.MatchedBy(requestWithURL("http://source-controller/chart.tgz"))).Return(response(http.StatusOK, archive), nil).Once()
		loaderService := NewHTTPChartArtifactLoader(client)

		first, firstErr := loaderService.GetChart(testCtx, "http://source-controller/chart.tgz", expectedDigest)
		second, secondErr := loaderService.GetChart(testCtx, "http://source-controller/chart.tgz", expectedDigest)

		require.NoError(t, firstErr)
		require.NoError(t, secondErr)
		assert.Equal(t, "test-chart", first.Name())
		assert.Equal(t, "test-chart", second.Name())
		assert.NotSame(t, first, second)
	})
}

func createTestChartArchive(t *testing.T) []byte {
	t.Helper()

	return createChartArchive(t, map[string]string{
		"test-chart/Chart.yaml":               "apiVersion: v2\nname: test-chart\nversion: 1.2.3\n",
		"test-chart/values.yaml":              "foo: bar\n",
		"test-chart/templates/configmap.yaml": "apiVersion: v1\nkind: ConfigMap\n",
	})
}

func createChartArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(content))}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return buffer.Bytes()
}

func requestWithURL(expectedURL string) func(*http.Request) bool {
	return func(request *http.Request) bool {
		return request.Method == http.MethodGet && request.URL.String() == expectedURL && request.Context() == context.Background()
	}
}

func response(status int, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(bytes.NewReader(body))}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}
