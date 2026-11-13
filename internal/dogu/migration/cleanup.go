package migration

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const volumeFieldManager = "k8s-dogu-operator-migration-cleanup"

// CleanupMigrationResources deletes the migration network policies, the migration secret and the migration volume claim of the dogu.
// The volume behind the claim and its storage are deleted by the provisioner.
//
// The claim is deleted last, because the migration counts as in progress as long as it exists.
// Otherwise, an interrupted cleanup would prepare the volume claim again.
func CleanupMigrationResources(ctx context.Context, k8sClient client.Client, dogu *v3beta1.Dogu) error {
	if err := k8sClient.DeleteAllOf(
		ctx,
		&networkingv1.NetworkPolicy{},
		client.InNamespace(dogu.Namespace),
		client.MatchingLabels(migrationLabels(dogu)),
	); err != nil {
		return fmt.Errorf("failed to delete migration network policies of dogu %q: %w", dogu.Spec.Name, err)
	}

	key := migrationResourceKey(dogu)
	objectMeta := metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name}
	if err := k8sClient.Delete(ctx, &corev1.Secret{ObjectMeta: objectMeta}); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete migration secret %s: %w", key, err)
	}

	if err := releaseMigrationVolume(ctx, k8sClient, key); err != nil {
		return fmt.Errorf("failed to release migration volume %s: %w", key, err)
	}

	if err := k8sClient.Delete(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: objectMeta}); client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete migration volume claim %s: %w", key, err)
	}

	return nil
}

// releaseMigrationVolume sets the reclaim policy of the volume bound to the migration volume claim to Delete.
// The migration retained the volume until the migration jobs took over its data.
func releaseMigrationVolume(ctx context.Context, k8sClient client.Client, key client.ObjectKey) error {
	claim := &corev1.PersistentVolumeClaim{}
	err := k8sClient.Get(ctx, key, claim)
	if apierrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get migration volume claim %s: %w", key, err)
	}

	if claim.Spec.VolumeName == "" {
		return nil
	}

	volume := corev1ac.PersistentVolume(claim.Spec.VolumeName).
		WithSpec(corev1ac.PersistentVolumeSpec().WithPersistentVolumeReclaimPolicy(corev1.PersistentVolumeReclaimDelete))
	if aErr := k8sClient.Apply(ctx, volume, client.FieldOwner(volumeFieldManager), client.ForceOwnership); aErr != nil {
		return fmt.Errorf("failed to set reclaim policy of migration volume %s to Delete: %w", claim.Spec.VolumeName, aErr)
	}

	return nil
}
