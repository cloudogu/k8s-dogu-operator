package charts

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

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	"github.com/opencontainers/go-digest"
	"helm.sh/helm/v3/pkg/chart/loader"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type InvalidChartArtifactError struct {
	cause error
}

func (err *InvalidChartArtifactError) Error() string {
	return fmt.Sprintf("invalid chart artifact: %v", err.cause)
}

func (err *InvalidChartArtifactError) Unwrap() error {
	return err.cause
}

func newInvalidChartArtifactError(err error) *InvalidChartArtifactError {
	return &InvalidChartArtifactError{cause: err}
}

type chartProvider struct {
	k8sClient  client.Client
	httpClient *http.Client
}

func NewChartProvider(k8sClient client.Client, httpClient *http.Client) ChartProvider {
	return &chartProvider{k8sClient: k8sClient, httpClient: httpClient}
}

func (provider *chartProvider) ArtifactDigest(ctx context.Context, doguResource *v3beta1.Dogu) (string, error) {
	repository, err := provider.getRepository(ctx, doguResource)
	if err != nil {
		return "", fmt.Errorf("failed to get repository: %w", err)
	}

	_, expectedDigest, _, err := getArtifactMetadata(repository)
	if err != nil {
		return "", fmt.Errorf("failed to get artifact metadata: %w", err)
	}

	return expectedDigest.String(), nil
}

func (provider *chartProvider) GetChartArchive(ctx context.Context, doguResource *v3beta1.Dogu) ([]byte, error) {
	repository, err := provider.getRepository(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	artifactURL, expectedDigest, expectedSize, err := getArtifactMetadata(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact metadata: %w", err)
	}

	archive, err := provider.downloadChart(ctx, artifactURL, expectedDigest, expectedSize)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %w", err)
	}

	return archive, nil
}

func (provider *chartProvider) getRepository(ctx context.Context, doguResource *v3beta1.Dogu) (*flux.OCIRepository, error) {
	repository := &flux.OCIRepository{}
	key := client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}
	if err := provider.k8sClient.Get(ctx, key, repository); err != nil {
		return nil, fmt.Errorf("failed to get OCIRepository %s: %w", key, err)
	}

	if repository.Status.ObservedGeneration != repository.Generation ||
		!metautil.IsStatusConditionTrue(repository.Status.Conditions, meta.ReadyCondition) {
		return nil, fmt.Errorf("OCIRepository %s is not ready for its current generation", key)
	}

	return repository, nil
}

func getArtifactMetadata(repository *flux.OCIRepository) (*url.URL, digest.Digest, *int64, error) {
	if repository.Status.Artifact == nil || repository.Status.Artifact.URL == "" || repository.Status.Artifact.Digest == "" {
		return nil, "", nil, fmt.Errorf("OCIRepository %s/%s has incomplete artifact metadata", repository.Namespace, repository.Name)
	}

	artifactURL, err := url.ParseRequestURI(repository.Status.Artifact.URL)
	if err != nil || artifactURL.Host == "" || (artifactURL.Scheme != "http" && artifactURL.Scheme != "https") {
		return nil, "", nil, newInvalidChartArtifactError(fmt.Errorf("invalid artifact URL %q", repository.Status.Artifact.URL))
	}

	expectedDigest, err := digest.Parse(repository.Status.Artifact.Digest)
	if err != nil {
		return nil, "", nil, newInvalidChartArtifactError(fmt.Errorf("invalid artifact digest: %w", err))
	}

	return artifactURL, expectedDigest, repository.Status.Artifact.Size, nil
}

func (provider *chartProvider) downloadChart(ctx context.Context, artifactURL *url.URL, expectedDigest digest.Digest, expectedSize *int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL.String(), nil)
	if err != nil {
		return nil, newInvalidChartArtifactError(fmt.Errorf("failed to create artifact HTTP request: %w", err))
	}

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute artifact HTTP GET request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}

	// We must buffer the whole archive because the Service caches these bytes and re-parses a fresh
	// chart per use. Bound that buffer: flux publishes the exact artifact size on the OCIRepository
	// status, so cap the read there when present, otherwise fall back to Helm's maximum chart size.
	// Read one byte past the cap so an over-sized body is detected rather than silently truncated.
	maxSize := loader.MaxDecompressedChartSize
	if expectedSize != nil && *expectedSize > 0 {
		maxSize = *expectedSize
	}

	archive, err := io.ReadAll(io.LimitReader(response.Body, maxSize+1))
	if err != nil {
		return nil, newInvalidChartArtifactError(fmt.Errorf("failed to read chart artifact: %w", err))
	}

	if int64(len(archive)) > maxSize {
		return nil, newInvalidChartArtifactError(fmt.Errorf("chart artifact exceeds maximum size of %d bytes", maxSize))
	}

	if actualDigest := expectedDigest.Algorithm().FromBytes(archive); actualDigest != expectedDigest {
		return nil, newInvalidChartArtifactError(errors.New("artifact digest mismatch"))
	}

	// Pre-Validate the chart returned to fail fast before putting the chart bytes into the cache.
	// The Service parses a fresh chart per use from  these bytes so dependency processing never mutates a shared instance.
	if _, err := loader.LoadArchive(bytes.NewReader(archive)); err != nil {
		return nil, newInvalidChartArtifactError(err)
	}

	return archive, nil
}
