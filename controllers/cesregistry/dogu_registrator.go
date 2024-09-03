package cesregistry

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	doguDescriptors "github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/errors"
)

type doguDescriptorRepo interface {
	Add(context.Context, doguDescriptors.SimpleDoguName, *core.Dogu) error
	DeleteAll(context.Context, doguDescriptors.SimpleDoguName) error
}

type doguVersionRegistry interface {
	GetCurrent(context.Context, doguDescriptors.SimpleDoguName) (doguDescriptors.DoguVersion, error)
	Enable(context.Context, doguDescriptors.DoguVersion) error
}

// CesDoguRegistrator is responsible for register dogus in the cluster
type CesDoguRegistrator struct {
	client               client.Client
	localDoguDescriptors doguDescriptorRepo
	doguVersions         doguVersionRegistry
}

// NewCESDoguRegistrator creates a new instance of the dogu registrator. It registers dogus in the dogu registry and
// generates keypairs
func NewCESDoguRegistrator(
	client client.Client,
	localDoguDescriptors doguDescriptorRepo,
	doguVersions doguVersionRegistry,
) *CesDoguRegistrator {
	return &CesDoguRegistrator{
		client:               client,
		localDoguDescriptors: localDoguDescriptors,
		doguVersions:         doguVersions,
	}
}

// RegisterNewDogu registers a completely new dogu in a cluster. Use RegisterDoguVersion() for upgrades of an existing
// dogu.
func (c *CesDoguRegistrator) RegisterNewDogu(ctx context.Context, _ *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)
	_, err := c.doguVersions.GetCurrent(ctx, doguDescriptors.SimpleDoguName(dogu.GetSimpleName()))
	enabled := !errors.IsNotFoundError(err)
	if err != nil && enabled {
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
	_, err := c.doguVersions.GetCurrent(ctx, doguDescriptors.SimpleDoguName(dogu.GetSimpleName()))
	enabled := !errors.IsNotFoundError(err)
	if err != nil && enabled {
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
func (c *CesDoguRegistrator) UnregisterDogu(ctx context.Context, doguName doguDescriptors.SimpleDoguName) error {
	err := c.localDoguDescriptors.DeleteAll(ctx, doguName)
	if err != nil {
		return fmt.Errorf("failed to unregister doguName %s: %w", doguName, err)
	}

	return nil
}

func (c *CesDoguRegistrator) enableDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	version, err := core.ParseVersion(dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to parse version of dogu %q: %w", dogu.GetSimpleName(), err)
	}

	err = c.doguVersions.Enable(ctx, doguDescriptors.DoguVersion{
		Name:    doguDescriptors.SimpleDoguName(dogu.GetSimpleName()),
		Version: version,
	})
	if err != nil {
		return fmt.Errorf("failed to enable dogu: %w", err)
	}
	return nil
}

func (c *CesDoguRegistrator) registerDoguInRegistry(ctx context.Context, dogu *core.Dogu) error {
	err := c.localDoguDescriptors.Add(ctx, doguDescriptors.SimpleDoguName(dogu.GetSimpleName()), dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu %s: %w", dogu.GetSimpleName(), err)
	}
	return nil
}
