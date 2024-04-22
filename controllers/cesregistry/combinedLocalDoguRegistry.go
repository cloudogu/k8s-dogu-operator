package cesregistry

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/exp/maps"

	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
)

// LocalDoguRegistry abstracts accessing various backends for reading and writing dogu specs (dogu.json).
type LocalDoguRegistry interface {
	// Enable makes the dogu spec reachable.
	Enable(ctx context.Context, dogu *core.Dogu) error
	// Register adds the given dogu spec to the local registry.
	Register(ctx context.Context, dogu *core.Dogu) error
	// UnregisterAllVersions deletes all versions of the dogu spec from the local registry and makes the spec unreachable.
	UnregisterAllVersions(ctx context.Context, name QualifiedDoguName) error
	// Reregister adds the new dogu spec to the local registry, enables it, and deletes all specs referenced by the old dogu name.
	// This is used for namespace changes and may contain an empty implementation if this action is not necessary.
	Reregister(ctx context.Context, oldName QualifiedDoguName, newDogu *core.Dogu) error
	// GetCurrent retrieves the spec of the referenced dogu's currently installed version.
	GetCurrent(ctx context.Context, name QualifiedDoguName) (*core.Dogu, error)
	// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions.
	GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error)
	// IsEnabled checks if the current spec of the referenced dogu is reachable.
	IsEnabled(ctx context.Context, name QualifiedDoguName) (bool, error)
}

// QualifiedDoguName is the full name of the dogu including the namespace.
type QualifiedDoguName struct {
	// Namespace is namespace the dogu is published under.
	Namespace string
	// SimpleName is the name of the dogu without the namespace.
	SimpleName string
}

// QualifiedName returns the QualifiedDoguName of the given dogu.
func QualifiedName(dogu *core.Dogu) QualifiedDoguName {
	return QualifiedDoguName{
		Namespace:  dogu.GetNamespace(),
		SimpleName: dogu.GetSimpleName(),
	}
}

// String returns namespace and simple name of the dogu, separated by a '/'.
func (qdn QualifiedDoguName) String() string {
	return fmt.Sprintf("%s/%s", qdn.Namespace, qdn.SimpleName)
}

// CombinedLocalDoguRegistry combines the ClusterNativeLocalDoguRegistry and EtcdLocalDoguRegistry for backwards-compatability reasons.
type CombinedLocalDoguRegistry struct {
	cnRegistry   *ClusterNativeLocalDoguRegistry
	etcdRegistry *EtcdLocalDoguRegistry
}

// Enable makes the dogu spec reachable.
func (cr *CombinedLocalDoguRegistry) Enable(ctx context.Context, dogu *core.Dogu) error {
	cnErr := cr.cnRegistry.Enable(ctx, dogu)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to enable dogu %q in cluster-native local registry: %w", dogu.GetSimpleName(), cnErr)
	}

	etcdErr := cr.etcdRegistry.Enable(ctx, dogu)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to enable dogu %q in ETCD local registry (legacy): %w", dogu.GetSimpleName(), etcdErr)
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
func (cr *CombinedLocalDoguRegistry) UnregisterAllVersions(ctx context.Context, name QualifiedDoguName) error {
	cnErr := cr.cnRegistry.UnregisterAllVersions(ctx, name)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to unregister dogu %q in cluster-native local registry: %w", name, cnErr)
	}

	etcdErr := cr.etcdRegistry.UnregisterAllVersions(ctx, name)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to unregister dogu %q in ETCD local registry (legacy): %w", name.SimpleName, etcdErr)
	}

	return errors.Join(cnErr, etcdErr)
}

// Reregister adds the new dogu spec to the local registry, enables it, and deletes all specs referenced by the old dogu name.
func (cr *CombinedLocalDoguRegistry) Reregister(ctx context.Context, oldName QualifiedDoguName, newDogu *core.Dogu) error {
	cnErr := cr.cnRegistry.Reregister(ctx, oldName, newDogu)
	if cnErr != nil {
		cnErr = fmt.Errorf("failed to reregister dogu %q in cluster-native local registry: %w", newDogu.GetSimpleName(), cnErr)
	}
	etcdErr := cr.etcdRegistry.Reregister(ctx, oldName, newDogu)
	if etcdErr != nil {
		etcdErr = fmt.Errorf("failed to reregister dogu %q in ETCD local registry (legacy): %w", newDogu.GetSimpleName(), etcdErr)
	}

	return errors.Join(cnErr, etcdErr)
}

// GetCurrent retrieves the spec of the referenced dogu's currently installed version.
func (cr *CombinedLocalDoguRegistry) GetCurrent(ctx context.Context, name QualifiedDoguName) (*core.Dogu, error) {
	logger := log.FromContext(ctx).
		WithName("CombinedLocalDoguRegistry.GetCurrent").
		WithValues("dogu.name", name.SimpleName).
		WithValues("dogu.namespace", name.Namespace)

	dogu, err := cr.cnRegistry.GetCurrent(ctx, name)
	if k8sErrs.IsNotFound(err) {
		logger.Error(err, "current dogu.json not found in cluster-native local registry; falling back to ETCD")

		dogu, err = cr.etcdRegistry.GetCurrent(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get current dogu.json of %q from ETCD local registry (legacy/fallback): %w", name, err)
		}

	} else if err != nil {
		return nil, fmt.Errorf("failed to get current dogu.json of %q from cluster-native local registry: %w", name, err)
	}

	return dogu, nil
}

// GetCurrentOfAll retrieves the specs of all dogus' currently installed versions.
func (cr *CombinedLocalDoguRegistry) GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error) {
	cmDogus, cnErr := cr.cnRegistry.GetCurrentOfAll(ctx)
	etcdDogus, etcdErr := cr.etcdRegistry.GetCurrentOfAll(ctx)
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
func (cr *CombinedLocalDoguRegistry) IsEnabled(ctx context.Context, name QualifiedDoguName) (bool, error) {
	logger := log.FromContext(ctx).
		WithName("CombinedLocalDoguRegistry.IsEnabled").
		WithValues("dogu.name", name.SimpleName).
		WithValues("dogu.namespace", name.Namespace)

	enabled, err := cr.cnRegistry.IsEnabled(ctx, name)
	if err != nil {
		return false, fmt.Errorf("failed to check if dogu %q is enabled in cluster-native local registry: %w", name, err)
	}

	if !enabled {
		logger.Error(err, "dogu is not enabled in cluster-native local registry; checking ETCD as fallback")
		enabled, err = cr.etcdRegistry.IsEnabled(ctx, name)
		if err != nil {
			return false, fmt.Errorf("failed to check if dogu %q is enabled in ETCD local registry (legacy): %w", name.SimpleName, err)
		}
	}

	return enabled, nil
}
