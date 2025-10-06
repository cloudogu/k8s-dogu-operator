package dependency

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
)

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	cesregistry.LocalDoguFetcher
}

// Validator checks if all necessary dependencies of the dogu are installed.
type Validator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}
