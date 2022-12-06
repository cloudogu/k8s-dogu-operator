package internal

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
)

// ServiceAccountCreator includes functionality to create necessary service accounts for a dogu.
type ServiceAccountCreator interface {
	// CreateAll is used to create all necessary service accounts for the given dogu.
	CreateAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

// ServiceAccountRemover includes functionality to remove existing service accounts for a dogu.
type ServiceAccountRemover interface {
	// RemoveAll is used to remove all existing service accounts for the given dogu.
	RemoveAll(ctx context.Context, dogu *cesappcore.Dogu) error
}
