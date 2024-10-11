package dependency

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
)

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName string) (installedDogu *cesappcore.Dogu, err error)
}
