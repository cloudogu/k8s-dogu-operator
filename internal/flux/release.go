// Package flux adapts the Flux CD resources the operator depends on to the operator's own domain
// types, keeping the Flux API (helm-controller, source-controller) contained behind a small seam.
package flux

import (
	"context"
	"fmt"

	helmflux "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReleaseOperation mirrors the .Release.* fields Helm sets while templating a chart. It describes
// the operation that will next be applied to a dogu's chart.
type ReleaseOperation struct {
	IsInstall bool
	IsUpgrade bool
	Revision  int
}

// installOperation is the default when nothing has been installed yet: a first install at revision 1,
// matching the state Helm produces for `helm install`.
func installOperation() ReleaseOperation {
	return ReleaseOperation{IsInstall: true, Revision: 1}
}

// HelmReleaseReader resolves the pending Helm release operation for a dogu by reading its HelmRelease.
type HelmReleaseReader struct {
	client client.Client
}

// NewHelmReleaseReader creates a HelmReleaseReader backed by the given cluster client.
func NewHelmReleaseReader(c client.Client) *HelmReleaseReader {
	return &HelmReleaseReader{client: c}
}

// ResolveOperation derives the pending release operation for the dogu's chart from its HelmRelease:
// - no HelmRelease, or one that has not been installed yet => install at revision 1
// - an installed HelmRelease at revision N => upgrade at revision N+1
//
// The HelmRelease is looked up by the dogu's own name and namespace, mirroring the OCIRepository
// naming convention used elsewhere for a dogu's chart artifact.
func (r *HelmReleaseReader) ResolveOperation(ctx context.Context, name, namespace string) (ReleaseOperation, error) {
	release := &helmflux.HelmRelease{}
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, release)
	if apierrors.IsNotFound(err) {
		return installOperation(), nil
	}

	if err != nil {
		return ReleaseOperation{}, fmt.Errorf("failed to get HelmRelease %s/%s: %w", namespace, name, err)
	}

	latest := release.Status.History.Latest()
	if latest == nil || latest.Version < 1 {
		return installOperation(), nil
	}

	return ReleaseOperation{IsUpgrade: true, Revision: latest.Version + 1}, nil
}
