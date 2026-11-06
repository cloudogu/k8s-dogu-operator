package pvc

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func growRequest(name, desired string) ResizeRequest {
	return ResizeRequest{PVC: client.ObjectKey{Namespace: testNamespace, Name: name}, Current: resource.MustParse("1Gi"), Desired: resource.MustParse(desired)}
}

func failingApplyFor(claimName string, err error) interceptor.Funcs {
	return interceptor.Funcs{Apply: func(ctx context.Context, c client.WithWatch, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
		if *obj.(*corev1ac.PersistentVolumeClaimApplyConfiguration).GetName() == claimName {
			return err
		}
		return c.Apply(ctx, obj, opts...)
	}}
}

func TestGrower_Grow(t *testing.T) {
	t.Run("raises the storage request of each claim", func(t *testing.T) {
		k8s := fake.NewClientBuilder().WithObjects(newPVC("data", "1Gi", nil), newPVC("logs", "1Gi", nil)).WithReturnManagedFields().Build()

		grown, err := NewGrower(k8s).Grow(context.Background(), []ResizeRequest{growRequest("data", "2Gi"), growRequest("logs", "3Gi")})

		require.NoError(t, err)
		require.Len(t, grown, 2)
		assert.Equal(t, GrownClaim{PVC: client.ObjectKey{Namespace: testNamespace, Name: "data"}, Previous: resource.MustParse("1Gi"), Desired: resource.MustParse("2Gi")}, grown[0])
		stored := &corev1.PersistentVolumeClaim{}
		require.NoError(t, k8s.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: "logs"}, stored))
		assert.Equal(t, resource.MustParse("3Gi"), stored.Spec.Resources.Requests[corev1.ResourceStorage])
		assert.True(t, slices.ContainsFunc(stored.ManagedFields, func(entry metav1.ManagedFieldsEntry) bool {
			return entry.Manager == growerFieldOwner && entry.Operation == metav1.ManagedFieldsOperationApply
		}))
	})

	for name, rejection := range map[string]error{
		"forbidden": apierrors.NewForbidden(corev1.Resource("persistentvolumeclaims"), "logs", errors.New("storageclass must support resize")),
		"invalid":   apierrors.NewInvalid(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim").GroupKind(), "logs", nil),
	} {
		t.Run("stops at a "+name+" claim and reports it as rejected", func(t *testing.T) {
			k8s := fake.NewClientBuilder().WithObjects(newPVC("data", "1Gi", nil), newPVC("logs", "1Gi", nil), newPVC("cache", "1Gi", nil)).
				WithInterceptorFuncs(failingApplyFor("logs", rejection)).Build()

			grown, err := NewGrower(k8s).Grow(context.Background(), []ResizeRequest{growRequest("data", "2Gi"), growRequest("logs", "2Gi"), growRequest("cache", "2Gi")})

			var rejected *GrowRejectedError
			require.ErrorAs(t, err, &rejected)
			assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "logs"}, rejected.PVC)
			assert.ErrorIs(t, err, rejection)
			assert.ErrorContains(t, err, `failed to grow PVC "ecosystem/logs" to 2Gi`)
			require.Len(t, grown, 1)
			assert.Equal(t, "data", grown[0].PVC.Name)
		})
	}

	t.Run("returns other apply errors unclassified", func(t *testing.T) {
		k8s := fake.NewClientBuilder().WithObjects(newPVC("data", "1Gi", nil)).
			WithInterceptorFuncs(failingApplyFor("data", apierrors.NewTimeoutError("apply timed out", 1))).Build()

		grown, err := NewGrower(k8s).Grow(context.Background(), []ResizeRequest{growRequest("data", "2Gi")})

		var rejected *GrowRejectedError
		assert.NotErrorAs(t, err, &rejected)
		assert.ErrorContains(t, err, "apply timed out")
		assert.Empty(t, grown)
	})
}
