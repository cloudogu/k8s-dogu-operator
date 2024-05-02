package localregistry

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/exp/maps"

	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
)

// CombinedLocalDoguRegistry combines the clusterNativeLocalDoguRegistry and etcdLocalDoguRegistry for backwards-compatability reasons.
type CombinedLocalDoguRegistry struct {
	cnRegistry   LocalDoguRegistry
	etcdRegistry LocalDoguRegistry
}

func NewCombinedLocalDoguRegistry(configMapClient v1.ConfigMapInterface, etcdRegistry registry.Registry) *CombinedLocalDoguRegistry {
	return &CombinedLocalDoguRegistry{
		cnRegistry: &clusterNativeLocalDoguRegistry{
			configMapClient: configMapClient,
		},
		etcdRegistry: &etcdLocalDoguRegistry{
			registry:     etcdRegistry,
			etcdRegistry: etcdRegistry.DoguRegistry(),
		}}
}

// Enable makes the dogu spec reachable.
func (cr *CombinedLocalDoguRegistry) Enable(ctx context.Context, dogu *core.Dogu) error {
	cnErr := cr.cnRegistry.Enable(ctx, dogu)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to enable dogu %q in cluster-native local registry: %w", dogu.Name, cnErr)
	}

	etcdErr := cr.etcdRegistry.Enable(ctx, dogu)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to enable dogu %q in ETCD local registry (legacy): %w", dogu.Name, etcdErr)
	}

	return errors.Join(cnErr, etcdErr)
}

// Register adds the given dogu spec to the local registry.
func (cr *CombinedLocalDoguRegistry) Register(ctx context.Context, dogu *core.Dogu) error {
	cnErr := cr.cnRegistry.Register(ctx, dogu)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to register dogu %q in cluster-native local registry: %w", dogu.Name, cnErr)
	}

	etcdErr := cr.etcdRegistry.Register(ctx, dogu)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to register dogu %q in ETCD local registry (legacy): %w", dogu.Name, etcdErr)
	}

	return errors.Join(cnErr, etcdErr)
}

// UnregisterAllVersions deletes all versions of the dogu spec from the local registry and makes the spec unreachable.
func (cr *CombinedLocalDoguRegistry) UnregisterAllVersions(ctx context.Context, simpleDoguName string) error {
	cnErr := cr.cnRegistry.UnregisterAllVersions(ctx, simpleDoguName)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to unregister dogu %q in cluster-native local registry: %w", simpleDoguName, cnErr)
	}

	etcdErr := cr.etcdRegistry.UnregisterAllVersions(ctx, simpleDoguName)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to unregister dogu %q in ETCD local registry (legacy): %w", simpleDoguName, etcdErr)
	}

	return errors.Join(cnErr, etcdErr)
}

// GetCurrent retrieves the spec of the referenced dogu's currently installed version.
func (cr *CombinedLocalDoguRegistry) GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error) {
	logger := log.FromContext(ctx).
		WithName("CombinedLocalDoguRegistry.GetCurrent").
		WithValues("dogu.name", simpleDoguName)

	dogu, err := cr.cnRegistry.GetCurrent(ctx, simpleDoguName)
	if k8sErrs.IsNotFound(err) {
		logger.Error(err, "current dogu.json not found in cluster-native local registry; falling back to ETCD")

		dogu, err = cr.etcdRegistry.GetCurrent(ctx, simpleDoguName)
		if err != nil {
			return nil, fmt.Errorf("failed to get current dogu.json of %q from ETCD local registry (legacy/fallback): %w", simpleDoguName, err)
		}

	} else if err != nil {
		return nil, fmt.Errorf("failed to get current dogu.json of %q from cluster-native local registry: %w", simpleDoguName, err)
	}

	return dogu, nil
}

// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions.
func (cr *CombinedLocalDoguRegistry) GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error) {
	cmDogus, cnErr := cr.cnRegistry.GetCurrentOfAll(ctx)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to get all current dogu.jsons from cluster-native local registry: %w", cnErr)
	}

	etcdDogus, etcdErr := cr.etcdRegistry.GetCurrentOfAll(ctx)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to get all current dogu.jsons from ETCD local registry (legacy): %w", etcdErr)
	}

	if err := errors.Join(cnErr, etcdErr); err != nil {
		return nil, err
	}

	return mergeSlices(cmDogus, etcdDogus, func(dogu *core.Dogu) string {
		return dogu.Name
	}), nil
}

func mergeSlices[T any, U comparable](slice1, slice2 []T, keyFn func(T) U) []T {
	combinedMap := make(map[U]T, len(slice1)+len(slice2))
	for _, t := range slice2 {
		combinedMap[keyFn(t)] = t
	}
	for _, t := range slice1 {
		combinedMap[keyFn(t)] = t
	}

	return maps.Values(combinedMap)
}

// IsEnabled checks if the current spec of the referenced dogu is reachable.
func (cr *CombinedLocalDoguRegistry) IsEnabled(ctx context.Context, simpleDoguName string) (bool, error) {
	logger := log.FromContext(ctx).
		WithName("CombinedLocalDoguRegistry.IsEnabled").
		WithValues("dogu.name", simpleDoguName)

	enabled, err := cr.cnRegistry.IsEnabled(ctx, simpleDoguName)
	if k8sErrs.IsNotFound(err) {
		logger.Error(err, "cluster-native local registry not found; checking ETCD as fallback")

		enabled, err = cr.etcdRegistry.IsEnabled(ctx, simpleDoguName)
		if err != nil {
			return false, fmt.Errorf("failed to check if dogu %q is enabled in ETCD local registry (legacy/fallback): %w", simpleDoguName, err)
		}

	} else if err != nil {
		return false, fmt.Errorf("failed to check if dogu %q is enabled in cluster-native local registry: %w", simpleDoguName, err)
	}

	return enabled, nil
}
