package cesregistry

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type doguVersionRegistry interface {
	cescommons.VersionRegistry
}

type localDoguDescriptorRepository interface {
	cescommons.LocalDoguDescriptorRepository
}

type remoteDoguDescriptorRepository interface {
	cescommons.RemoteDoguDescriptorRepository
}

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
	// FetchForResource fetches the dogu descriptor for the desired version of the dogu from the local dogu registry.
	FetchForResource(ctx context.Context, doguResource *k8sv2.Dogu) (*cesappcore.Dogu, error)
}

// ResourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type ResourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *k8sv2.Dogu) (*cesappcore.Dogu, *k8sv2.DevelopmentDoguMap, error)
}

// DoguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type DoguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

type K8sClient interface {
	client.Client
}
