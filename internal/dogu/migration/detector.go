package migration

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configTypeLabel     = "k8s.cloudogu.com/type"
	sensitiveConfigType = "sensitive-config"

	managedByLabel = "app.kubernetes.io/managed-by"
	managedByHelm  = "Helm"
)

// State describes whether a dogu needs a migration.
type State int

const (
	// NotRequired means the dogu is installed or changed the regular V3 way.
	NotRequired State = iota
	// Required means V2 resources of the dogu exist and have to be migrated.
	Required
	// InProgress means only migration resources are left, so an interrupted migration has to be resumed.
	InProgress
)

// Detector derives the migration State of a dogu from the resources present in the cluster.
type Detector struct {
	client client.Reader
}

// NewDetector creates a Detector.
func NewDetector(c client.Reader) *Detector {
	return &Detector{client: c}
}

// Detect returns the migration State of the given dogu.
func (d *Detector) Detect(ctx context.Context, dogu *v3beta1.Dogu) (State, error) {
	if dogu.Spec.DoguApiVersion != v3beta1.DoguApiVersionV3 {
		return NotRequired, nil
	}

	found, err := d.hasSensitiveConfig(ctx, dogu)
	if err != nil {
		return NotRequired, err
	}
	if found {
		return Required, nil
	}

	found, err = d.hasVolumeClaim(ctx, dogu)
	if err != nil {
		return NotRequired, err
	}
	if found {
		return Required, nil
	}

	found, err = d.hasMigrationResources(ctx, dogu)
	if err != nil {
		return NotRequired, err
	}
	if found {
		return InProgress, nil
	}

	return NotRequired, nil
}

// hasMigrationResources reports whether the migration secret or the migration volume claim of the dogu exists.
func (d *Detector) hasMigrationResources(ctx context.Context, dogu *v3beta1.Dogu) (bool, error) {
	found, err := d.get(ctx, migrationResourceKey(dogu), &corev1.Secret{})
	if err != nil || found {
		return found, err
	}

	return d.get(ctx, migrationResourceKey(dogu), &corev1.PersistentVolumeClaim{})
}

// migrationResourceKey names the temporary resources that hold the V2 resources' content during a migration.
func migrationResourceKey(dogu *v3beta1.Dogu) client.ObjectKey {
	return client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name + "-migration"}
}

// hasVolumeClaim reports whether the V2 data PVC of the dogu exists.
func (d *Detector) hasVolumeClaim(ctx context.Context, dogu *v3beta1.Dogu) (bool, error) {
	claim := &corev1.PersistentVolumeClaim{}
	found, err := d.get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, claim)
	if err != nil || !found {
		return false, err
	}

	return claim.Labels[managedByLabel] != managedByHelm, nil
}

// hasSensitiveConfig reports whether the V2 sensitive config secret of the dogu exists.
func (d *Detector) hasSensitiveConfig(ctx context.Context, dogu *v3beta1.Dogu) (bool, error) {
	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: dogu.Namespace, Name: strings.ToLower(dogu.Spec.Name + "-config")}
	found, err := d.get(ctx, key, secret)
	if err != nil || !found {
		return false, err
	}

	return secret.Labels[configTypeLabel] == sensitiveConfigType, nil
}

// get fetches the object and reports whether it exists. Any error but NotFound is returned.
func (d *Detector) get(ctx context.Context, key client.ObjectKey, obj client.Object) (bool, error) {
	err := d.client.Get(ctx, key, obj)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get %T %s: %w", obj, key, err)
	}

	return true, nil
}
