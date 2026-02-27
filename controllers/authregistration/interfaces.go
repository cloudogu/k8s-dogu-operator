package authregistration

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	authRegClientV1 "github.com/cloudogu/k8s-auth-registration-lib/client/typed/api/v1"
)

// AuthRegistrationManager describes the AuthRegistration lifecycle for a dogu.
type Manager interface {
	// EnsureAuthRegistration creates/updates the AuthRegistration and syncs sensitive credentials.
	EnsureAuthRegistration(ctx context.Context, dogu *cesappcore.Dogu) error
	// RemoveAuthRegistration removes the AuthRegistration belonging to the given dogu.
	RemoveAuthRegistration(ctx context.Context, doguName cescommons.SimpleName) error
}

type authRegistrationClient interface {
	authRegClientV1.AuthRegistrationInterface
}
