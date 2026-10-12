package charts

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

type ChartProvider interface {
	// ArtifactDigest returns the current OCIRepository artifact digest for the dogu without
	// downloading the artifact.
	ArtifactDigest(ctx context.Context, doguResource *v3beta1.Dogu) (string, error)
	// GetChartArchive downloads the ready artifact, verifies its digest and that it is a well-formed
	// Helm chart archive, and returns the raw archive bytes. The Service caches these bytes and parses
	// a fresh, independently mutable chart per use.
	GetChartArchive(ctx context.Context, doguResource *v3beta1.Dogu) ([]byte, error)
}
