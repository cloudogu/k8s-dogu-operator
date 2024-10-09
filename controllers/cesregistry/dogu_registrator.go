package cesregistry

import (
	"context"
	"fmt"
	regLibDogu "github.com/cloudogu/k8s-registry-lib/dogu"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/v2/api/v1"
)

// CesDoguRegistrator is responsible for register dogus in the cluster
type CesDoguRegistrator struct {
	versionRegistry doguVersionRegistry
	doguRepository  localDoguDescriptorRepository
}

// NewCESDoguRegistrator creates a new instance of the dogu registrator. It registers dogus in the dogu registry and
// generates keypairs
func NewCESDoguRegistrator(doguVersionRegistry regLibDogu.DoguVersionRegistry, doguDescriptorRepo regLibDogu.LocalDoguDescriptorRepository) *CesDoguRegistrator {
	return &CesDoguRegistrator{
		versionRegistry: doguVersionRegistry,
		doguRepository:  doguDescriptorRepo,
	}
}

// RegisterNewDogu registers a completely new dogu in a cluster. Use RegisterDoguVersion() for upgrades of an existing
// dogu.
func (c *CesDoguRegistrator) RegisterNewDogu(ctx context.Context, _ *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	enabled, _, err := checkDoguVersionEnabled(ctx, c.versionRegistry, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is enabled: %w", err)
	}

	if enabled {
		logger.Info("Skipping dogu registration because it is already installed and enabled in the dogu registry")
		return nil
	}

	coreVersion, err := dogu.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get dogu-version for dogu '%s' with version '%s': %w", dogu.GetSimpleName(), dogu.Version, err)
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	return c.enableDoguInRegistry(ctx, regLibDogu.DoguVersion{
		Name:    regLibDogu.SimpleDoguName(dogu.GetSimpleName()),
		Version: coreVersion,
	})
}

// RegisterDoguVersion registers an upgrade of an existing dogu in a cluster. Use RegisterNewDogu() to complete new
// dogu installations.
func (c *CesDoguRegistrator) RegisterDoguVersion(ctx context.Context, dogu *core.Dogu) error {

	enabled, currentDoguVersion, err := checkDoguVersionEnabled(ctx, c.versionRegistry, dogu.GetSimpleName())
	if err != nil {
		return fmt.Errorf("failed to check if dogu is enabled: %w", err)
	}

	if !enabled {
		return fmt.Errorf("could not register dogu version: previous version not found")
	}

	err = c.registerDoguInRegistry(ctx, dogu)
	if err != nil {
		return err
	}

	return c.enableDoguInRegistry(ctx, currentDoguVersion)
}

// UnregisterDogu deletes a dogu from the dogu registry
func (c *CesDoguRegistrator) UnregisterDogu(ctx context.Context, doguName string) error {
	err := c.doguRepository.DeleteAll(ctx, regLibDogu.SimpleDoguName(doguName))
	if err != nil {
		return fmt.Errorf("failed to unregister dogu %s: %w", doguName, err)
	}

	return nil
}

func (c *CesDoguRegistrator) enableDoguInRegistry(ctx context.Context, doguVersion regLibDogu.DoguVersion) error {
	err := c.versionRegistry.Enable(ctx, doguVersion)
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}
	return nil
}

func (c *CesDoguRegistrator) registerDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.doguRepository.Add(ctx, regLibDogu.SimpleDoguName(dogu.GetSimpleName()), dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu %s: %w", dogu.GetSimpleName(), err)
	}
	return nil
}
