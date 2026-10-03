package charts

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service is the dogu-keyed facade over chart rendering/validation. It is the seam both the
// validation step and the PVC-resize step depend on. It owns a per-dogu load cache (keyed by the
// dogu's NamespacedName, invalidated when the artifact digest changes)
type Service struct {
	loader     ChartProvider
	restConfig *rest.Config
	caps       *chartutil.Capabilities

	mu     sync.Mutex
	charts map[client.ObjectKey]*chart
}

// NewService creates a Service. caps should be discovered once from the cluster (KubeVersion +
// API versions) so local renders gate API versions the same way Flux will.
func NewService(loader ChartProvider, restConfig *rest.Config, caps *chartutil.Capabilities) *Service {
	return &Service{
		loader:     loader,
		restConfig: restConfig,
		caps:       caps,
		charts:     make(map[client.ObjectKey]*chart),
	}
}

// Render renders the dogu's chart server-side against the cluster and returns the uninterpreted
// objects. The result is shared and MUST be treated read-only by callers.
func (s *Service) Render(ctx context.Context, doguResource *v3beta1.Dogu, values map[string]any) ([]*unstructured.Unstructured, error) {
	c, err := s.chartFor(ctx, doguResource)
	if err != nil {
		return nil, err
	}

	return c.render(values, s.restConfig)
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
	delete(s.charts, key)
	s.mu.Unlock()
}

// chartFor returns the cached chart for the dogu, reloading it when the artifact digest has changed.
func (s *Service) chartFor(ctx context.Context, doguResource *v3beta1.Dogu) (*chart, error) {
	key := client.ObjectKeyFromObject(doguResource)

	digest, err := s.loader.ArtifactDigest(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart artifact digest: %w", err)
	}

	s.mu.Lock()
	if cached, ok := s.charts[key]; ok && cached.digest == digest {
		s.mu.Unlock()

		return cached, nil
	}

	// unlock until the chart is loaded
	s.mu.Unlock()

	raw, err := s.loader.GetChart(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	c := &chart{
		raw:    raw,
		ref:    releaseRefFor(doguResource),
		caps:   s.caps,
		digest: digest,
	}

	s.mu.Lock()
	s.charts[key] = c
	s.mu.Unlock()

	return c, nil
}

// releaseRefFor derives the Helm release identity for a dogu.
func releaseRefFor(doguResource *v3beta1.Dogu) ReleaseRef {
	return ReleaseRef{Name: doguResource.Name, Namespace: doguResource.Namespace}
}
