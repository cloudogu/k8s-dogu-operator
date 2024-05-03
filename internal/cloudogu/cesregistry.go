package cloudogu

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName string) (installedDogu *cesappcore.Dogu, err error)
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
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

// LocalDoguRegistry abstracts accessing various backends for reading and writing dogu specs (dogu.json).
type LocalDoguRegistry interface {
	// Enable makes the dogu spec reachable.
	Enable(ctx context.Context, dogu *cesappcore.Dogu) error
	// Register adds the given dogu spec to the local registry.
	Register(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterAllVersions deletes all versions of the dogu spec from the local registry and makes the spec unreachable.
	UnregisterAllVersions(ctx context.Context, simpleDoguName string) error
	// Reregister adds the new dogu spec to the local registry, enables it, and deletes all specs referenced by the old dogu name.
	// This is used for namespace changes and may contain an empty implementation if this action is not necessary.
	Reregister(ctx context.Context, newDogu *cesappcore.Dogu) error
	// GetCurrent retrieves the spec of the referenced dogu's currently installed version.
	GetCurrent(ctx context.Context, simpleDoguName string) (*cesappcore.Dogu, error)
	// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions.
	GetCurrentOfAll(ctx context.Context) ([]*cesappcore.Dogu, error)
	// IsEnabled checks if the current spec of the referenced dogu is reachable.
	IsEnabled(ctx context.Context, simpleDoguName string) (bool, error)
}
