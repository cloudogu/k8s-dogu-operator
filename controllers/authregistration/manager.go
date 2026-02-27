package authregistration

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type AuthRegistrationManager struct{}

// NewManager creates an AuthRegistrationManager which can be used to create and remove AuthRegistration resources.
func NewManager() *AuthRegistrationManager {
	return &AuthRegistrationManager{}
}

// EnsureAuthRegistration creates/updates the AuthRegistration and syncs sensitive credentials.
func (sm *AuthRegistrationManager) EnsureAuthRegistration(ctx context.Context, dogu *cesappcore.Dogu) error {
	if dogu == nil {
		return fmt.Errorf("dogu must not be nil")
	}

	log.FromContext(ctx).V(1).Info(
		"AuthRegistration stub invoked",
		"dogu", dogu.GetSimpleName(),
	)
	return nil
}

// RemoveAuthRegistration removes the AuthRegistration belonging to the given dogu.
func (sm *AuthRegistrationManager) RemoveAuthRegistration(ctx context.Context, doguName cescommons.SimpleName) error {
	log.FromContext(ctx).V(1).Info(
		"AuthRegistration removal stub invoked",
		"dogu", doguName.String(),
	)
	return nil
}
