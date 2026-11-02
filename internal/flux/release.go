// Package flux adapts the Flux CD resources the operator depends on to the operator's own domain
// types, keeping the Flux API (helm-controller, source-controller) contained behind a small seam.
package flux

import (
	"context"
	"fmt"

	helmflux "github.com/fluxcd/helm-controller/api/v2"
	sdk "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
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

// HelmReleaseReader reads a dogu's HelmRelease and its stored Helm release.
type HelmReleaseReader struct {
	client    client.Client
	clientSet kubernetes.Interface
}

// NewHelmReleaseReader creates a HelmReleaseReader backed by the given cluster client.
func NewHelmReleaseReader(c client.Client, clientSet kubernetes.Interface) *HelmReleaseReader {
	return &HelmReleaseReader{client: c, clientSet: clientSet}
}

// StoredRelease reads the values, manifest and status of the latest revision in Flux's history.
func (r *HelmReleaseReader) StoredRelease(hr *helmflux.HelmRelease) (*sdk.Release, error) {
	// The latest revision may have failed; the pending comparison checks its status separately.
	latest := hr.Status.History.Latest()
	if latest == nil {
		return nil, nil
	}
	// Helm decodes the release secret in the storage namespace, which can differ from the workload namespace.
	store := storage.Init(driver.NewSecrets(r.clientSet.CoreV1().Secrets(hr.GetStorageNamespace())))
	stored, err := store.Get(latest.Name, latest.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to read stored Helm release %s/%s revision %d: %w", hr.GetStorageNamespace(), latest.Name, latest.Version, err)
	}
	return stored, nil
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
