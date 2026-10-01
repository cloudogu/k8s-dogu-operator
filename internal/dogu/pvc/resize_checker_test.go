package pvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "ecosystem"

func TestResizeChecker_CheckStandalonePVCs(t *testing.T) {
	ctx := context.Background()
	fast := "fast"

	t.Run("returns an empty result when there are no rendered PVCs", func(t *testing.T) {
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, nil)

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("ignores a rendered PVC that does not exist yet", func(t *testing.T) {
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, []client.Object{newPVC("data", "2Gi", nil)})

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("treats equivalent quantities and a defaulted storage class as unchanged", func(t *testing.T) {
		live := newPVC("data", "1024Mi", &fast)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, []client.Object{newPVC("data", "1Gi", nil)})

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("returns only PVCs that require expansion", func(t *testing.T) {
		liveA := newPVC("data-a", "1Gi", nil)
		liveA.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("4Gi")}
		liveB := newPVC("data-b", "2Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(liveA, liveB).Build())

		result, err := checker.Check(ctx, []client.Object{
			newPVC("data-b", "2Gi", nil),
			newPVC("data-a", "2Gi", nil),
		})

		require.NoError(t, err)
		require.Len(t, result.ResizeRequests, 1)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data-a"}, result.ResizeRequests[0].PVC)
		assert.Equal(t, "2Gi", result.ResizeRequests[0].Desired.String())
		assert.NoError(t, result.ResizeRequests[0].Err)
	})

	t.Run("returns a typed shrink error on its PVC request", func(t *testing.T) {
		live := newPVC("data", "10Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, []client.Object{newPVC("data", "2Gi", nil)})

		require.NoError(t, err)
		require.Len(t, result.ResizeRequests, 1)
		var shrinkError *VolumeShrinkError
		require.ErrorAs(t, result.ResizeRequests[0].Err, &shrinkError)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, shrinkError.PVC)
		assert.Equal(t, "10Gi", shrinkError.Current.String())
		assert.Equal(t, "2Gi", shrinkError.Desired.String())
	})
}

func TestResizeChecker_CheckStorageClass(t *testing.T) {
	ctx := context.Background()
	fast := "fast"
	slow := "slow"
	empty := ""

	tests := []struct {
		name         string
		desiredClass *string
		currentClass *string
		wantError    bool
	}{
		{name: "both unset", desiredClass: nil, currentClass: nil},
		{name: "desired unset accepts a defaulted live class", desiredClass: nil, currentClass: &fast},
		{name: "same explicit class", desiredClass: &fast, currentClass: &fast},
		{name: "different explicit class", desiredClass: &slow, currentClass: &fast, wantError: true},
		{name: "explicit desired class and unset live class", desiredClass: &fast, currentClass: nil, wantError: true},
		{name: "explicit empty desired class is not unset", desiredClass: &empty, currentClass: nil, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			live := newPVC("data", "1Gi", tt.currentClass)
			checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

			result, err := checker.Check(ctx, []client.Object{newPVC("data", "1Gi", tt.desiredClass)})

			require.NoError(t, err)
			if !tt.wantError {
				assert.Empty(t, result.ResizeRequests)
				return
			}

			require.Len(t, result.ResizeRequests, 1)
			var storageClassError *StorageClassImmutableError
			require.ErrorAs(t, result.ResizeRequests[0].Err, &storageClassError)
			assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, storageClassError.PVC)
			assert.Equal(t, tt.currentClass, storageClassError.Current)
			assert.Equal(t, tt.desiredClass, storageClassError.Desired)
		})
	}
}

func TestResizeChecker_CheckErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a technical error separately when the live PVC cannot be read", func(t *testing.T) {
		readError := apierrors.NewForbidden(schema.GroupResource{Resource: "persistentvolumeclaims"}, "data", errors.New("denied"))
		checker := NewResizeChecker(errorReader{err: readError})

		result, err := checker.Check(ctx, []client.Object{newPVC("data", "2Gi", nil)})

		require.ErrorIs(t, err, readError)
		assert.ErrorContains(t, err, "failed to read live PVC \"ecosystem/data\"")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("returns a technical error when the live PVC has no storage request", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Spec.Resources.Requests = nil
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, []client.Object{newPVC("data", "2Gi", nil)})

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" has no storage request")
		assert.Empty(t, result.ResizeRequests)
	})
}

func TestExtractDesiredClaims(t *testing.T) {
	t.Run("extracts standalone PVCs and ignores unrelated objects", func(t *testing.T) {
		pvc := newPVC("data", "2Gi", nil)
		configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: testNamespace}}

		claims, err := extractDesiredClaims([]client.Object{configMap, pvc})

		require.NoError(t, err)
		require.Len(t, claims, 1)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, claims[0].key)
		assert.Zero(t, claims[0].storage.Cmp(resource.MustParse("2Gi")))
	})

	t.Run("rejects a rendered PVC without a storage request", func(t *testing.T) {
		pvc := newPVC("data", "1Gi", nil)
		pvc.Spec.Resources.Requests = nil

		claims, err := extractDesiredClaims([]client.Object{pvc})

		require.ErrorContains(t, err, "rendered PVC \"ecosystem/data\" has no storage request")
		assert.Empty(t, claims)
	})
}

func newPVC(name, storage string, storageClass *string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(storage)},
			},
		},
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return r.err
}

func (r errorReader) List(context.Context, client.ObjectList, ...client.ListOption) error {
	return r.err
}
