package cesregistry

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
)

// CesDoguRegistrator is responsible for register dogus in the cluster
type CesDoguRegistrator struct {
	client            client.Client
	registry          cesregistry.Registry
	localDoguRegistry thirdParty.LocalDoguRegistry
}

// NewCESDoguRegistrator creates a new instance of the dogu registrator. It registers dogus in the dogu registry and
// generates keypairs
func NewCESDoguRegistrator(
	client client.Client,
	localDoguRegistry thirdParty.LocalDoguRegistry,
	registry cesregistry.Registry,
) *CesDoguRegistrator {
	return &CesDoguRegistrator{
		client:            client,
		registry:          registry,
		localDoguRegistry: localDoguRegistry,
	}
}

// RegisterNewDogu registers a completely new dogu in a cluster. Use RegisterDoguVersion() for upgrades of an existing
// dogu.
func (c *CesDoguRegistrator) RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	enabled, err := c.localDoguRegistry.IsEnabled(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is already installed and enabled: %w", err)
	}

	if enabled {
		logger.Info("Skipping dogu registration because it is already installed and enabled in the dogu registry")
		return nil
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	return c.enableDoguInRegistry(ctx, dogu)
}

// RegisterDoguVersion registers an upgrade of an existing dogu in a cluster. Use RegisterNewDogu() to complete new
// dogu installations.
func (c *CesDoguRegistrator) RegisterDoguVersion(ctx context.Context, dogu *core.Dogu) error {
	enabled, err := c.localDoguRegistry.IsEnabled(ctx, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is already installed and enabled: %w", err)
	}

	if !enabled {
		return fmt.Errorf("could not register dogu version: previous version not found")
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	return c.enableDoguInRegistry(ctx, dogu)
}

// UnregisterDogu deletes a dogu from the dogu registry
func (c *CesDoguRegistrator) UnregisterDogu(ctx context.Context, dogu string) error {
	err := c.localDoguRegistry.UnregisterAllVersions(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to unregister dogu %s: %w", dogu, err)
	}

	return nil
}

func (c *CesDoguRegistrator) enableDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.localDoguRegistry.Enable(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}
	return nil
}

func (c *CesDoguRegistrator) registerDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.localDoguRegistry.Register(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu %s: %w", dogu.GetSimpleName(), err)
	}
	return nil
}
