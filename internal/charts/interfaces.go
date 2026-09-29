package charts

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	helmchart "helm.sh/helm/v3/pkg/chart"
)

type ChartProvider interface {
	// ArtifactDigest returns the current OCIRepository artifact digest for the dogu without
	// downloading the artifact.
	ArtifactDigest(ctx context.Context, doguResource *v3beta1.Dogu) (string, error)
	// GetChart downloads and unpacks the ready artifact into a raw Helm chart.
	GetChart(ctx context.Context, doguResource *v3beta1.Dogu) (*helmchart.Chart, error)
}
