package chart

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace   = "ecosystem"
	testDoguName    = "postgresql"
	testArtifactURL = "http://source-controller.flux-system.svc/chart.tgz"
)

func TestChartProviderGetChart(t *testing.T) {
	archive := createTestChartArchive(t)
	repository := readyRepository(archive)
	doguResource := testDoguResource()
	body := &trackingReadCloser{Reader: bytes.NewReader(archive)}
	requests := 0
	provider := newTestProvider(t, repository, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, testArtifactURL, request.URL.String())
		return response(http.StatusOK, body), nil
	}))

	actual, err := provider.GetChart(context.Background(), doguResource)

	require.NoError(t, err)
	require.NotNil(t, actual)
	assert.Equal(t, "test-chart", actual.Name())
	assert.Equal(t, "bar", actual.Values["foo"])
	require.Len(t, actual.Templates, 1)
	assert.Equal(t, "templates/configmap.yaml", actual.Templates[0].Name)
	assert.Equal(t, 1, requests)
	assert.True(t, body.closed)
}

func TestChartProviderRejectsUnavailableRepositoriesBeforeDownloading(t *testing.T) {
	archive := createTestChartArchive(t)
	tests := []struct {
		name       string
		repository *flux.OCIRepository
	}{
		{name: "missing repository"},
		{name: "stale repository status", repository: func() *flux.OCIRepository {
			repository := readyRepository(archive)
			repository.Status.ObservedGeneration--
			return repository
		}()},
		{name: "repository not ready", repository: func() *flux.OCIRepository {
			repository := readyRepository(archive)
			repository.Status.Conditions[0].Status = metav1.ConditionFalse
			return repository
		}()},
		{name: "missing artifact", repository: func() *flux.OCIRepository {
			repository := readyRepository(archive)
			repository.Status.Artifact = nil
			return repository
		}()},
		{name: "missing artifact URL", repository: func() *flux.OCIRepository {
			repository := readyRepository(archive)
			repository.Status.Artifact.URL = ""
			return repository
		}()},
		{name: "missing artifact digest", repository: func() *flux.OCIRepository {
			repository := readyRepository(archive)
			repository.Status.Artifact.Digest = ""
			return repository
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			provider := newTestProvider(t, tt.repository, roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return nil, assert.AnError
			}))

			actual, err := provider.GetChart(context.Background(), testDoguResource())

			assert.Nil(t, actual)
			require.Error(t, err)
			assert.False(t, isInvalidChartArtifactError(err))
			assert.Zero(t, requests)
		})
	}
}

func TestChartProviderRejectsInvalidMetadataBeforeDownloading(t *testing.T) {
	archive := createTestChartArchive(t)
	tests := []struct {
		name         string
		artifactURL  string
		artifactHash string
	}{
		{name: "invalid URL", artifactURL: "file:///tmp/chart.tgz", artifactHash: digest.SHA256.FromBytes(archive).String()},
		{name: "invalid digest", artifactURL: testArtifactURL, artifactHash: "not-a-digest"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := readyRepository(archive)
			repository.Status.Artifact.URL = tt.artifactURL
			repository.Status.Artifact.Digest = tt.artifactHash
			requests := 0
			provider := newTestProvider(t, repository, roundTripFunc(func(*http.Request) (*http.Response, error) {
				requests++
				return nil, assert.AnError
			}))

			actual, err := provider.GetChart(context.Background(), testDoguResource())

			assert.Nil(t, actual)
			require.True(t, isInvalidChartArtifactError(err))
			assert.Zero(t, requests)
		})
	}
}

func TestChartProviderRejectsDigestMismatch(t *testing.T) {
	archive := createTestChartArchive(t)
	repository := readyRepository(archive)
	repository.Status.Artifact.Digest = digest.SHA256.FromString("different").String()
	provider := newTestProvider(t, repository, responseTransport(http.StatusOK, archive))

	actual, err := provider.GetChart(context.Background(), testDoguResource())

	assert.Nil(t, actual)
	require.True(t, isInvalidChartArtifactError(err))
	assert.ErrorContains(t, err, "digest mismatch")
}

func TestChartProviderIncludesTrailingBytesInDigest(t *testing.T) {
	archive := append(createTestChartArchive(t), []byte("trailing data")...)
	repository := readyRepository(archive)
	provider := newTestProvider(t, repository, responseTransport(http.StatusOK, archive))

	actual, err := provider.GetChart(context.Background(), testDoguResource())

	require.NoError(t, err)
	assert.Equal(t, "test-chart", actual.Name())
}

func TestChartProviderRejectsInvalidArchive(t *testing.T) {
	archive := []byte("not a chart")
	repository := readyRepository(archive)
	provider := newTestProvider(t, repository, responseTransport(http.StatusOK, archive))

	actual, err := provider.GetChart(context.Background(), testDoguResource())

	assert.Nil(t, actual)
	require.True(t, isInvalidChartArtifactError(err))
}

func TestChartProviderClassifiesHTTPStatus(t *testing.T) {
	archive := createTestChartArchive(t)
	tests := []struct {
		status  int
		invalid bool
	}{
		{status: http.StatusNotFound},
		{status: http.StatusBadRequest, invalid: true},
		{status: http.StatusRequestTimeout},
		{status: http.StatusTooManyRequests},
		{status: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			provider := newTestProvider(t, readyRepository(archive), responseTransport(tt.status, nil))

			actual, err := provider.GetChart(context.Background(), testDoguResource())

			assert.Nil(t, actual)
			require.Error(t, err)
			if tt.invalid {
				require.True(t, isInvalidChartArtifactError(err))
			} else {
				assert.False(t, isInvalidChartArtifactError(err))
			}
		})
	}
}

func TestChartProviderReturnsTransportAndReadErrors(t *testing.T) {
	archive := createTestChartArchive(t)
	t.Run("transport error", func(t *testing.T) {
		provider := newTestProvider(t, readyRepository(archive), roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, assert.AnError
		}))

		actual, err := provider.GetChart(context.Background(), testDoguResource())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, assert.AnError)
		assert.False(t, isInvalidChartArtifactError(err))
	})

	t.Run("body read error", func(t *testing.T) {
		readErr := errors.New("read failed")
		body := &trackingReadCloser{Reader: &failingReader{data: archive[:len(archive)/2], err: readErr}}
		provider := newTestProvider(t, readyRepository(archive), roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusOK, body), nil
		}))

		actual, err := provider.GetChart(context.Background(), testDoguResource())

		assert.Nil(t, actual)
		assert.ErrorIs(t, err, readErr)
		require.True(t, isInvalidChartArtifactError(err))
		assert.True(t, body.closed)
	})
}

func TestChartProviderUsesHelmArchiveSizeLimit(t *testing.T) {
	archive := createChartArchive(t, map[string][]byte{
		"test-chart/Chart.yaml": []byte("apiVersion: v2\nname: test-chart\nversion: 1.0.0\n"),
		"test-chart/large":      bytes.Repeat([]byte("x"), 5*1024*1024+1),
	})
	provider := newTestProvider(t, readyRepository(archive), responseTransport(http.StatusOK, archive))

	actual, err := provider.GetChart(context.Background(), testDoguResource())

	assert.Nil(t, actual)
	require.True(t, isInvalidChartArtifactError(err))
	assert.ErrorContains(t, err, "larger than the maximum file size")
}

func isInvalidChartArtifactError(err error) bool {
	var invalidChartArtifactError *InvalidChartArtifactError
	return errors.As(err, &invalidChartArtifactError)
}

func newTestProvider(t *testing.T, repository *flux.OCIRepository, transport http.RoundTripper) ChartProvider {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, flux.AddToScheme(scheme))
	builder := fake.NewClientBuilder().WithScheme(scheme)
	if repository != nil {
		builder = builder.WithObjects(repository)
	}
	return NewChartProvider(builder.Build(), &http.Client{Transport: transport})
}

func testDoguResource() *v3beta1.Dogu {
	return &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "resource-name-differs", Namespace: testNamespace},
		Spec:       v3beta1.DoguSpec{Name: testDoguName},
	}
}

func readyRepository(archive []byte) *flux.OCIRepository {
	return &flux.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{Name: testDoguName, Namespace: testNamespace, Generation: 2},
		Status: flux.OCIRepositoryStatus{
			ObservedGeneration: 2,
			Conditions: []metav1.Condition{{
				Type:   meta.ReadyCondition,
				Status: metav1.ConditionTrue,
			}},
			Artifact: &meta.Artifact{URL: testArtifactURL, Digest: digest.SHA256.FromBytes(archive).String()},
		},
	}
}

func createTestChartArchive(t *testing.T) []byte {
	t.Helper()
	return createChartArchive(t, map[string][]byte{
		"test-chart/Chart.yaml":               []byte("apiVersion: v2\nname: test-chart\nversion: 1.2.3\n"),
		"test-chart/values.yaml":              []byte("foo: bar\n"),
		"test-chart/templates/configmap.yaml": []byte("apiVersion: v1\nkind: ConfigMap\n"),
	})
}

func createChartArchive(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		size := int64(len(content))
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o600, Size: size}))
		_, err := tarWriter.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return buffer.Bytes()
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func response(statusCode int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: statusCode, Status: http.StatusText(statusCode), Body: body}
}

func responseTransport(statusCode int, body []byte) http.RoundTripper {
	return roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(statusCode, io.NopCloser(bytes.NewReader(body))), nil
	})
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

type failingReader struct {
	data []byte
	err  error
}

func (reader *failingReader) Read(target []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, reader.err
	}
	read := copy(target, reader.data)
	reader.data = reader.data[read:]
	return read, nil
}
