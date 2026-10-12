package charts

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// cachedArtifact holds a downloaded chart archive and the artifact digest it was fetched for.
type cachedArtifact struct {
	data   []byte
	digest string
}

type releaseResolver interface {
	ResolveOperation(ctx context.Context, name, namespace string) (flux.ReleaseOperation, error)
}

// Service is the dogu-keyed facade over the dogu chart. It owns a per-dogu cache of the downloaded
// chart archive bytes (keyed by the dogu's NamespacedName, invalidated when the artifact digest
// changes). Each access parses a fresh chart from the cached bytes so that dependency processing,
// which mutates the chart in place, never corrupts a shared instance.
type Service struct {
	loader     ChartProvider
	restConfig *rest.Config
	caps       *chartutil.Capabilities
	resolver   releaseResolver

	mu        sync.Mutex
	artifacts map[client.ObjectKey]cachedArtifact
}

// NewService creates a Service.
func NewService(loader ChartProvider, restConfig *rest.Config, caps *chartutil.Capabilities, resolver releaseResolver) *Service {
	return &Service{
		loader:     loader,
		restConfig: restConfig,
		caps:       caps,
		resolver:   resolver,
		artifacts:  make(map[client.ObjectKey]cachedArtifact),
	}
}

// Render renders the dogu's chart server-side against the cluster and returns the uninterpreted objects.
func (s *Service) Render(ctx context.Context, doguResource *v3beta1.Dogu, values map[string]any) ([]*unstructured.Unstructured, error) {
	op, err := s.resolver.ResolveOperation(ctx, doguResource.Spec.Name, doguResource.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve release operation: %w", err)
	}

	c, err := s.chartFor(ctx, doguResource)
	if err != nil {
		return nil, err
	}

	return c.render(values, s.restConfig, op)
}

// ValidateValues validates the values against the chart's values.schema.json only. It performs no
// rendering and needs no cluster access.
func (s *Service) ValidateValues(ctx context.Context, doguResource *v3beta1.Dogu, values map[string]any) error {
	c, err := s.chartFor(ctx, doguResource)
	if err != nil {
		return err
	}

	return c.validateValues(values)
}

// ChartPatchTemplate returns the raw chart patch template file from the chart, if present. The bool
// reports whether the file exists.
func (s *Service) ChartPatchTemplate(ctx context.Context, doguResource *v3beta1.Dogu) ([]byte, bool, error) {
	c, err := s.chartFor(ctx, doguResource)
	if err != nil {
		return nil, false, err
	}

	tmpFile, ok := c.getChartPatchTpl()

	return tmpFile, ok, nil
}

// DoguMetaValues returns the raw dogu-meta-values file from the chart, if present. The bool
// reports whether the file exists.
func (s *Service) DoguMetaValues(ctx context.Context, doguResource *v3beta1.Dogu) ([]byte, bool, error) {
	c, err := s.chartFor(ctx, doguResource)
	if err != nil {
		return nil, false, err
	}

	tmpFile, ok := c.getDoguValuesMeta()

	return tmpFile, ok, nil
}

// Evict drops a dogu's cached chart. Call it when the delete-finalizer runs so the cache does not grow
// unbounded across the operator's lifetime.
func (s *Service) Evict(doguResource *v3beta1.Dogu) {
	key := client.ObjectKeyFromObject(doguResource)

	s.mu.Lock()
	delete(s.artifacts, key)
	s.mu.Unlock()
}

// chartFor returns a freshly parsed chart for the dogu. The underlying archive bytes are cached and
// only re-downloaded when the artifact digest changes, but the chart is parsed new on every call so
// callers receive an independently mutable instance (dependency processing mutates it in place).
func (s *Service) chartFor(ctx context.Context, doguResource *v3beta1.Dogu) (*chart, error) {
	key := client.ObjectKeyFromObject(doguResource)

	digest, err := s.loader.ArtifactDigest(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart artifact digest: %w", err)
	}

	data, err := s.artifactBytes(ctx, doguResource, key, digest)
	if err != nil {
		return nil, err
	}

	raw, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse chart archive: %w", err)
	}

	return &chart{
		raw:  raw,
		ref:  releaseRefFor(doguResource),
		caps: s.caps,
	}, nil
}

// artifactBytes returns the cached chart archive bytes for the dogu, downloading and caching them
// when the cache is empty or the artifact digest has changed.
func (s *Service) artifactBytes(ctx context.Context, doguResource *v3beta1.Dogu, key client.ObjectKey, digest string) ([]byte, error) {
	s.mu.Lock()
	if cached, ok := s.artifacts[key]; ok && cached.digest == digest {
		s.mu.Unlock()

		return cached.data, nil
	}

	// Unlock while downloading chart
	s.mu.Unlock()

	data, err := s.loader.GetChartArchive(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart archive: %w", err)
	}

	s.mu.Lock()
	s.artifacts[key] = cachedArtifact{data: data, digest: digest}
	s.mu.Unlock()

	return data, nil
}

// releaseRefFor derives the Helm release identity for a dogu.
func releaseRefFor(doguResource *v3beta1.Dogu) ReleaseRef {
	return ReleaseRef{Name: doguResource.Name, Namespace: doguResource.Namespace}
}
