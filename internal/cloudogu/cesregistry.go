package cloudogu

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(doguName string) (installedDogu *cesappcore.Dogu, err error)
}

// ResourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type ResourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *k8sv1.Dogu) (*cesappcore.Dogu, *k8sv1.DevelopmentDoguMap, error)
}

// DoguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type DoguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(dogu string) error
}

// SecretResourceGenerator is used to generate kubernetes secret resources
type SecretResourceGenerator interface {
	// CreateDoguSecret generates a secret for the dogu resource containing the given data.
	CreateDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error)
}
