package postinstall

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}
