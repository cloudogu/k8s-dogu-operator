package authregistration

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type stubManager struct{}

// NewStubManager creates a placeholder AuthRegistration manager.
// The detailed implementation (CR creation and secret sync) is added in follow-up changes.
func NewStubManager() Manager {
	return &stubManager{}
}

func (sm *stubManager) EnsureAuthRegistration(ctx context.Context, dogu *cesappcore.Dogu) error {
	if dogu == nil {
		return fmt.Errorf("dogu must not be nil")
	}

	log.FromContext(ctx).V(1).Info(
		"AuthRegistration stub invoked",
		"dogu", dogu.GetSimpleName(),
	)
	return nil
}

func (sm *stubManager) RemoveAuthRegistration(ctx context.Context, doguName cescommons.SimpleName) error {
	log.FromContext(ctx).V(1).Info(
		"AuthRegistration removal stub invoked",
		"dogu", doguName.String(),
	)
	return nil
}
