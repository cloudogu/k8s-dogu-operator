package chart

import (
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
	digest "github.com/opencontainers/go-digest"
	helmchart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func (provider *chartProvider) GetChart(ctx context.Context, doguResource *v3beta1.Dogu) (*helmchart.Chart, error) {
	repository, err := provider.getRepository(ctx, doguResource)
	if err != nil {
		return nil, err
	}

	artifactURL, expectedDigest, err := getArtifactMetadata(repository)
	if err != nil {
		return nil, err
	}

	return provider.downloadChart(ctx, artifactURL, expectedDigest)
}

func (provider *chartProvider) getRepository(ctx context.Context, doguResource *v3beta1.Dogu) (*flux.OCIRepository, error) {
	repository := &flux.OCIRepository{}
	key := client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}
	if err := provider.k8sClient.Get(ctx, key, repository); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("OCIRepository %s not found: %w", key, err)
		}
		return nil, fmt.Errorf("failed to get OCIRepository %s: %w", key, err)
	}

	if repository.Status.ObservedGeneration != repository.Generation ||
		!metautil.IsStatusConditionTrue(repository.Status.Conditions, meta.ReadyCondition) {
		return nil, fmt.Errorf("OCIRepository %s is not ready for its current generation", key)
	}

	return repository, nil
}

func getArtifactMetadata(repository *flux.OCIRepository) (*url.URL, digest.Digest, error) {
	if repository.Status.Artifact == nil || repository.Status.Artifact.URL == "" || repository.Status.Artifact.Digest == "" {
		return nil, "", fmt.Errorf("OCIRepository %s/%s has incomplete artifact metadata", repository.Namespace, repository.Name)
	}

	artifactURL, err := url.ParseRequestURI(repository.Status.Artifact.URL)
	if err != nil || artifactURL.Host == "" || (artifactURL.Scheme != "http" && artifactURL.Scheme != "https") {
		return nil, "", newInvalidChartArtifactError(fmt.Errorf("invalid artifact URL %q", repository.Status.Artifact.URL))
	}

	expectedDigest, err := digest.Parse(repository.Status.Artifact.Digest)
	if err != nil {
		return nil, "", newInvalidChartArtifactError(fmt.Errorf("invalid artifact digest: %w", err))
	}

	return artifactURL, expectedDigest, nil
}

func (provider *chartProvider) downloadChart(ctx context.Context, artifactURL *url.URL, expectedDigest digest.Digest) (*helmchart.Chart, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL.String(), nil)
	if err != nil {
		return nil, newInvalidChartArtifactError(fmt.Errorf("failed to create artifact request: %w", err))
	}

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart artifact: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if err = validateResponseStatus(response); err != nil {
		return nil, err
	}

	verifier := expectedDigest.Verifier()
	reader := io.TeeReader(response.Body, verifier)
	loadedChart, err := loader.LoadArchive(reader)
	if err != nil {
		return nil, newInvalidChartArtifactError(err)
	}

	if _, err = io.Copy(verifier, response.Body); err != nil {
		return nil, fmt.Errorf("failed to finish reading chart artifact: %w", err)
	}
	if !verifier.Verified() {
		return nil, newInvalidChartArtifactError(errors.New("artifact digest mismatch"))
	}

	return loadedChart, nil
}

func validateResponseStatus(response *http.Response) error {
	switch response.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("chart artifact not ready: unexpected HTTP status %s", response.Status)
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return fmt.Errorf("failed to download chart artifact: unexpected HTTP status %s", response.Status)
	default:
		if response.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("failed to download chart artifact: unexpected HTTP status %s", response.Status)
		}
		return newInvalidChartArtifactError(fmt.Errorf("unexpected HTTP status %s", response.Status))
	}
}
