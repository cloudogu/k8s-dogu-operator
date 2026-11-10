package migration

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func migrationNetworkPolicy(name, doguName string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		Name:      name,
		Namespace: testNamespace,
		Labels:    map[string]string{"k8s.cloudogu.com/app": "ces", "k8s.cloudogu.com/type": "migration", "k8s.cloudogu.com/dogu.name": doguName}}
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes the migration resources of the dogu only", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(
			&corev1.Secret{Name: "nexus-migration", Namespace: testNamespace},
			pvc("nexus-migration", nil),
			migrationNetworkPolicy("nexus-migration-postgresql", "nexus"),
			migrationNetworkPolicy("cas-migration-postgresql", "cas"),
		).Build()

		require.NoError(t, CleanupMigrationResources(ctx, cl, v3Dogu("nexus")))

		assertGone(t, cl, "nexus-migration", &corev1.Secret{})
		assertGone(t, cl, "nexus-migration", &corev1.PersistentVolumeClaim{})
		assertGone(t, cl, "nexus-migration-postgresql", &networkingv1.NetworkPolicy{})
		assertExists(t, cl, "cas-migration-postgresql", &networkingv1.NetworkPolicy{})
		require.NoError(t, cl.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: "cas-migration-postgresql"}, &networkingv1.NetworkPolicy{}))
	})

	t.Run("lets the provisioner delete the volume of the migration volume claim", func(t *testing.T) {
		claim := pvc("nexus-migration", nil)
		claim.Spec.VolumeName = "pv-1"
		volume := &corev1.PersistentVolume{
			Name: "pv-1",
			Spec: corev1.PersistentVolumeSpec{PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain},
		}
		// The deduced type converter is needed, because the default one fails to apply persistent volumes.
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).WithTypeConverters(managedfields.NewDeducedTypeConverter()).WithObjects(claim, volume).Build()

		require.NoError(t, CleanupMigrationResources(ctx, cl, v3Dogu("nexus")))

		assertGone(t, cl, "nexus-migration", &corev1.PersistentVolumeClaim{})
		require.NoError(t, cl.Get(ctx, client.ObjectKey{Name: "pv-1"}, volume))
		assert.Equal(t, corev1.PersistentVolumeReclaimDelete, volume.Spec.PersistentVolumeReclaimPolicy)
	})

	t.Run("succeeds without migration resources", func(t *testing.T) {
		cl := fake.NewClientBuilder().WithScheme(testScheme(t)).Build()

		require.NoError(t, CleanupMigrationResources(ctx, cl, v3Dogu("nexus")))
	})

	t.Run("cleanup fails", func(t *testing.T) {
		failDeleteOf := func(target client.Object) interceptor.Funcs {
			return interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				if reflect.TypeOf(obj) == reflect.TypeOf(target) {
					return assert.AnError
				}
				return c.Delete(ctx, obj, opts...)
			}}
		}

		errorTests := []struct {
			name    string
			funcs   interceptor.Funcs
			wantErr string
		}{
			{
				name: "network policies",
				funcs: interceptor.Funcs{DeleteAllOf: func(context.Context, client.WithWatch, client.Object, ...client.DeleteAllOfOption) error {
					return assert.AnError
				}},
				wantErr: `failed to delete migration network policies of dogu "nexus"`,
			},
			{
				name:    "secret",
				funcs:   failDeleteOf(&corev1.Secret{}),
				wantErr: "failed to delete migration secret ecosystem/nexus-migration",
			},
			{
				name: "volume claim lookup",
				funcs: interceptor.Funcs{Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
					return assert.AnError
				}},
				wantErr: "failed to get migration volume claim ecosystem/nexus-migration",
			},
			{
				name: "volume",
				funcs: interceptor.Funcs{Apply: func(context.Context, client.WithWatch, runtime.ApplyConfiguration, ...client.ApplyOption) error {
					return assert.AnError
				}},
				wantErr: "failed to set reclaim policy of migration volume pv-1 to Delete",
			},
			{
				name:    "volume claim",
				funcs:   failDeleteOf(&corev1.PersistentVolumeClaim{}),
				wantErr: "failed to delete migration volume claim ecosystem/nexus-migration",
			},
		}

		for _, tt := range errorTests {
			t.Run("fails to clean up the "+tt.name, func(t *testing.T) {
				claim := pvc("nexus-migration", nil)
				claim.Spec.VolumeName = "pv-1"
				cl := fake.NewClientBuilder().WithScheme(testScheme(t)).
					WithTypeConverters(managedfields.NewDeducedTypeConverter()).
					WithObjects(claim, &corev1.PersistentVolume{Name: "pv-1"}).
					WithInterceptorFuncs(tt.funcs).Build()

				err := CleanupMigrationResources(ctx, cl, v3Dogu("nexus"))

				require.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, tt.wantErr)
			})
		}
	})
}

func assertGone(t *testing.T, cl client.Client, name string, obj client.Object) {
	t.Helper()
	err := cl.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, obj)
	assert.True(t, apierrors.IsNotFound(err), "expected %T %s to be deleted, got %v", obj, name, err)
}

func assertExists(t *testing.T, cl client.Client, name string, obj client.Object) {
	t.Helper()
	err := cl.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: name}, obj)
	assert.NoError(t, err, "expected %T %s to exist, got %v", obj, name, err)
}
