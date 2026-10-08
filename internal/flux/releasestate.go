package flux

import (
	helmflux "github.com/fluxcd/helm-controller/api/v2"
)

// DesiredRelease describes the release state the operator expects the helm-controller to apply for a dogu.
type DesiredRelease struct {
	// ChartVersion is the chart version requested by the dogu resource.
	ChartVersion string
	// Generation is the generation of the HelmRelease spec the operator wrote last.
	// It changes with every spec change, e.g. a new chart version or new values.
	Generation int64
}

// NewDesiredRelease builds the desired release from the dogu's requested chart version and the current HelmRelease.
func NewDesiredRelease(chartVersion string, release *helmflux.HelmRelease) DesiredRelease {
	return DesiredRelease{
		ChartVersion: chartVersion,
		Generation:   release.Generation,
	}
}
