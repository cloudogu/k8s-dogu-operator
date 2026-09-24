package chart

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	helmchart "helm.sh/helm/v3/pkg/chart"
)

type ChartProvider interface {
	GetChart(ctx context.Context, doguResource *v3beta1.Dogu) (*helmchart.Chart, error)
}
