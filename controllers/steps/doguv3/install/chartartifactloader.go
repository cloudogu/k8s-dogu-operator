package install

import (
	"bytes"
	"context"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	digest "github.com/opencontainers/go-digest"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/utils/lru"
)

const (
	maxChartArtifactSize      int64 = 4 * 1024 * 1024
	defaultChartArtifactCache       = 3
)

var (
	errChartArtifactNotFound = errors.New("chart artifact not found")
	errInvalidChartArtifact  = errors.New("invalid chart artifact")
)

type HTTPChartArtifactLoader struct {
	httpClient HTTPClient
	cache      *lru.Cache
}

func NewHTTPChartArtifactLoader(httpClient HTTPClient) *HTTPChartArtifactLoader {
	return &HTTPChartArtifactLoader{
		httpClient: httpClient,
		cache:      lru.New(defaultChartArtifactCache),
	}
}

func (loaderService *HTTPChartArtifactLoader) GetChart(ctx context.Context, artifactURL, expectedDigest string) (*chart.Chart, error) {
	expected, err := digest.Parse(expectedDigest)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to validate artifact digest: %w", errInvalidChartArtifact, err)
	}

	parsedURL, err := url.ParseRequestURI(artifactURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("%w: failed to validate artifact URL %q", errInvalidChartArtifact, artifactURL)
	}

	if cached, ok := loaderService.cache.Get(expected.String()); ok {
		if archive, isArchive := cached.([]byte); isArchive {
			return loadChart(archive)
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create artifact request: %w", errInvalidChartArtifact, err)
	}
	response, err := loaderService.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart artifact: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if err = validateResponseStatus(response); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxChartArtifactSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read chart artifact: %w", err)
	}
	if int64(len(data)) > maxChartArtifactSize {
		return nil, fmt.Errorf("%w: failed to read chart artifact: size exceeds %d bytes", errInvalidChartArtifact, maxChartArtifactSize)
	}

	actual := expected.Algorithm().FromBytes(data)
	if actual != expected {
		return nil, fmt.Errorf("%w: failed to verify chart artifact: digest mismatch: expected %s, got %s", errInvalidChartArtifact, expected, actual)
	}

	loadedChart, err := loadChart(data)
	if err != nil {
		return nil, err
	}
	// Cache the verified archive so each caller receives a newly parsed chart.
	loaderService.cache.Add(expected.String(), data)

	return loadedChart, nil
}

func validateResponseStatus(response *http.Response) error {
	switch {
	case response.StatusCode == http.StatusOK:
		return nil
	case response.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: unexpected HTTP status %s", errChartArtifactNotFound, response.Status)
	case response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("failed to download chart artifact: unexpected HTTP status %s", response.Status)
	default:
		return fmt.Errorf("%w: unexpected HTTP status %s", errInvalidChartArtifact, response.Status)
	}
}

func loadChart(data []byte) (*chart.Chart, error) {
	loadedChart, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to load chart artifact: %w", errInvalidChartArtifact, err)
	}

	return loadedChart, nil
}
