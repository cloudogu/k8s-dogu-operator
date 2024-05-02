package localregistry

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
)

type etcdLocalDoguRegistry struct {
	registry     registry.Registry
	etcdRegistry registry.DoguRegistry
}

// Enable makes the dogu spec reachable
// by setting the current key in the ETCD to its currently installed version.
func (er *etcdLocalDoguRegistry) Enable(_ context.Context, dogu *core.Dogu) error {
	return er.etcdRegistry.Enable(dogu)
}

// Register adds the given dogu spec to the local registry in the ETCD.
func (er *etcdLocalDoguRegistry) Register(_ context.Context, dogu *core.Dogu) error {
	return er.etcdRegistry.Register(dogu)
}

// UnregisterAllVersions deletes all versions of the dogu spec from the local registry in the ETCD
// and makes the spec unreachable by deleting the current key in ETCD.
func (er *etcdLocalDoguRegistry) UnregisterAllVersions(_ context.Context, simpleDoguName string) error {
	err := er.registry.DoguConfig(simpleDoguName).RemoveAll()
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return fmt.Errorf("failed to remove dogu config: %w", err)
	}

	err = er.etcdRegistry.Unregister(simpleDoguName)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return fmt.Errorf("failed to unregister dogu %s: %w", simpleDoguName, err)
	}

	return nil
}

// GetCurrent retrieves the spec of the referenced dogu's currently installed version from ETCD.
func (er *etcdLocalDoguRegistry) GetCurrent(_ context.Context, simpleDoguName string) (*core.Dogu, error) {
	return er.etcdRegistry.Get(simpleDoguName)
}

// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions from ETCD.
func (er *etcdLocalDoguRegistry) GetCurrentOfAll(_ context.Context) ([]*core.Dogu, error) {
	return er.etcdRegistry.GetAll()
}

// IsEnabled checks if the current spec of the referenced dogu is reachable by checking if the current key is set in ETCD.
func (er *etcdLocalDoguRegistry) IsEnabled(_ context.Context, simpleDoguName string) (bool, error) {
	return er.etcdRegistry.IsEnabled(simpleDoguName)
}
