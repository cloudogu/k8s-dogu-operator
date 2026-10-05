package pvc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
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

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("treats equivalent quantities and a defaulted storage class as unchanged", func(t *testing.T) {
		live := newPVC("data", "1024Mi", &fast)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "1Gi", nil)))

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("returns only PVCs that require expansion", func(t *testing.T) {
		liveA := newPVC("data-a", "1Gi", nil)
		liveB := newPVC("data-b", "2Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(liveA, liveB).Build())

		result, err := checker.Check(ctx, renderedObjects(t,
			newPVC("data-b", "2Gi", nil),
			newPVC("data-a", "2Gi", nil),
		))

		require.NoError(t, err)
		require.Len(t, result.ResizeRequests, 1)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data-a"}, result.ResizeRequests[0].PVC)
		assert.Equal(t, "2Gi", result.ResizeRequests[0].Desired.String())
	})

	t.Run("rejects a live PVC whose request exceeds its capacity", func(t *testing.T) {
		live := newPVC("data", "20Gi", nil)
		live.Status.Capacity[corev1.ResourceStorage] = resource.MustParse("10Gi")
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "15Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" is not ready: storage request 20Gi exceeds capacity 10Gi")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("classifies an overprovisioned live PVC by request and capacity", func(t *testing.T) {
		tests := []struct {
			name        string
			desired     string
			wantRequest bool
			wantShrink  bool
		}{
			{name: "shrink below current request", desired: "5Gi", wantShrink: true},
			{name: "unchanged request", desired: "10Gi"},
			{name: "larger request within current capacity", desired: "15Gi", wantRequest: true},
			{name: "expansion beyond current capacity", desired: "25Gi", wantRequest: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				live := newPVC("data", "10Gi", nil)
				live.Status.Capacity[corev1.ResourceStorage] = resource.MustParse("20Gi")
				checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

				result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", tt.desired, nil)))

				if tt.wantShrink {
					var shrinkError *VolumeShrinkError
					require.ErrorAs(t, err, &shrinkError)
					assert.Equal(t, "10Gi", shrinkError.Current.String())
					assert.Empty(t, result.ResizeRequests)
					return
				}

				require.NoError(t, err)
				if !tt.wantRequest {
					assert.Empty(t, result.ResizeRequests)
					return
				}

				require.Len(t, result.ResizeRequests, 1)
				assert.Equal(t, tt.desired, result.ResizeRequests[0].Desired.String())
			})
		}
	})

	t.Run("returns a typed shrink error", func(t *testing.T) {
		live := newPVC("data", "10Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		var shrinkError *VolumeShrinkError
		require.ErrorAs(t, err, &shrinkError)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, shrinkError.PVC)
		assert.Equal(t, "10Gi", shrinkError.Current.String())
		assert.Equal(t, "2Gi", shrinkError.Desired.String())
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("returns all validation errors without partial expansion requests", func(t *testing.T) {
		fast := "fast"
		slow := "slow"
		liveExpand := newPVC("expand", "1Gi", nil)
		liveShrink := newPVC("shrink", "10Gi", nil)
		liveStorageClass := newPVC("storage-class", "1Gi", &fast)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(liveExpand, liveShrink, liveStorageClass).Build())

		result, err := checker.Check(ctx, renderedObjects(t,
			newPVC("expand", "2Gi", nil),
			newPVC("shrink", "2Gi", nil),
			newPVC("storage-class", "1Gi", &slow),
		))

		var shrinkError *VolumeShrinkError
		require.ErrorAs(t, err, &shrinkError)
		var storageClassError *StorageClassImmutableError
		require.ErrorAs(t, err, &storageClassError)
		assert.Empty(t, result.ResizeRequests)
	})
}

func TestResizeChecker_CheckStorageClass(t *testing.T) {
	ctx := context.Background()
	fast := "fast"
	slow := "slow"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			live := newPVC("data", "1Gi", tt.currentClass)
			checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

			result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "1Gi", tt.desiredClass)))

			if !tt.wantError {
				require.NoError(t, err)
				assert.Empty(t, result.ResizeRequests)
				return
			}

			var storageClassError *StorageClassImmutableError
			require.ErrorAs(t, err, &storageClassError)
			assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, storageClassError.PVC)
			assert.Equal(t, tt.currentClass, storageClassError.Current)
			assert.Equal(t, tt.desiredClass, storageClassError.Desired)
			assert.Empty(t, result.ResizeRequests)
		})
	}

	t.Run("rejects an empty desired storage class", func(t *testing.T) {
		empty := ""
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "1Gi", &empty)))

		require.ErrorContains(t, err, "rendered PVC \"ecosystem/data\" uses unsupported static provisioning")
		assert.Empty(t, result.ResizeRequests)
	})
}

func TestResizeChecker_CheckStatefulSetClaims(t *testing.T) {
	ctx := context.Background()

	t.Run("returns no partial result when a volume claim template is invalid", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", ptr.To[int32](2), 0,
			newPVC("data", "2Gi", nil),
			newPVC("logs", "1Gi", nil),
		)
		liveData0 := newPVC("data-postgres-0", "1Gi", nil)
		liveData1 := newPVC("data-postgres-1", "2Gi", nil)
		liveLogs0 := newPVC("logs-postgres-0", "2Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(liveData0, liveData1, liveLogs0).Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		var shrinkError *VolumeShrinkError
		require.ErrorAs(t, err, &shrinkError)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "logs-postgres-0"}, shrinkError.PVC)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("defaults nil replicas to one", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", nil, 0, newPVC("data", "2Gi", nil))
		live := newPVC("data-postgres-0", "1Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		require.NoError(t, err)
		require.Len(t, result.ResizeRequests, 1)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data-postgres-0"}, result.ResizeRequests[0].PVC)
	})

	t.Run("does not derive PVCs for zero replicas", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", ptr.To[int32](0), 0, newPVC("data", "2Gi", nil))
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		require.NoError(t, err)
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("uses the configured start ordinal", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", ptr.To[int32](2), 3, newPVC("data", "2Gi", nil))
		live3 := newPVC("data-postgres-3", "1Gi", nil)
		live4 := newPVC("data-postgres-4", "1Gi", nil)
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live3, live4).Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		require.NoError(t, err)
		require.Len(t, result.ResizeRequests, 2)
		requests := resizeRequestsByName(result.ResizeRequests)
		assert.Contains(t, requests, "data-postgres-3")
		assert.Contains(t, requests, "data-postgres-4")
	})

	t.Run("returns all volume claim template errors", func(t *testing.T) {
		missingStorage := newPVC("data", "1Gi", nil)
		missingStorage.Spec.Resources.Requests = nil
		emptyStorageClass := ""
		staticProvisioning := newPVC("logs", "1Gi", &emptyStorageClass)
		statefulSet := newStatefulSet("postgres", ptr.To[int32](1), 0, missingStorage, staticProvisioning)
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		require.ErrorContains(t, err, "rendered PVC \"ecosystem/data-postgres-0\" has no storage request")
		require.ErrorContains(t, err, "rendered PVC \"ecosystem/logs-postgres-0\" uses unsupported static provisioning")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("rejects negative replicas in rendered output", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", ptr.To[int32](-1), 0, newPVC("data", "2Gi", nil))
		checker := NewResizeChecker(fake.NewClientBuilder().Build())

		result, err := checker.Check(ctx, renderedObjects(t, statefulSet))

		require.ErrorContains(t, err, "rendered StatefulSet \"ecosystem/postgres\" has negative replicas")
		assert.Empty(t, result.ResizeRequests)
	})
}

func TestResizeChecker_CheckErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("returns a technical error separately when the live PVC cannot be read", func(t *testing.T) {
		readError := apierrors.NewForbidden(schema.GroupResource{Resource: "persistentvolumeclaims"}, "data", errors.New("denied"))
		checker := NewResizeChecker(errorReader{err: readError})

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorIs(t, err, readError)
		assert.ErrorContains(t, err, "failed to read live PVC \"ecosystem/data\"")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("returns a technical error when the live PVC has no storage request", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Spec.Resources.Requests = nil
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" has no storage request")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("rejects a live PVC that is not bound", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Status.Phase = corev1.ClaimPending
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" is not bound")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("rejects a live PVC with an active resize status", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Status.AllocatedResourceStatuses = map[corev1.ResourceName]corev1.ClaimResourceStatus{
			corev1.ResourceStorage: corev1.PersistentVolumeClaimControllerResizeInProgress,
		}
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" has storage resize status \"ControllerResizeInProgress\"")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("rejects a live PVC with an active resize condition", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Status.Conditions = []corev1.PersistentVolumeClaimCondition{{
			Type:   corev1.PersistentVolumeClaimResizing,
			Status: corev1.ConditionTrue,
		}}
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" has active resize condition \"Resizing\"")
		assert.Empty(t, result.ResizeRequests)
	})

	t.Run("rejects a live PVC without reported capacity", func(t *testing.T) {
		live := newPVC("data", "1Gi", nil)
		live.Status.Capacity = nil
		checker := NewResizeChecker(fake.NewClientBuilder().WithObjects(live).Build())

		result, err := checker.Check(ctx, renderedObjects(t, newPVC("data", "2Gi", nil)))

		require.ErrorContains(t, err, "live PVC \"ecosystem/data\" has no storage capacity")
		assert.Empty(t, result.ResizeRequests)
	})
}

func TestExtractDesiredClaims(t *testing.T) {
	t.Run("extracts standalone PVCs and ignores unrelated objects", func(t *testing.T) {
		pvc := newPVC("data", "2Gi", nil)
		configMap := &corev1.ConfigMap{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
			ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: testNamespace},
		}

		claims, err := extractDesiredClaims(renderedObjects(t, configMap, pvc))

		require.NoError(t, err)
		require.Len(t, claims, 1)
		assert.Equal(t, client.ObjectKey{Namespace: testNamespace, Name: "data"}, claims[0].key)
		assert.Zero(t, claims[0].storage.Cmp(resource.MustParse("2Gi")))
	})

	t.Run("rejects a standalone PVC without a namespace", func(t *testing.T) {
		pvc := newPVC("data", "1Gi", nil)
		pvc.Namespace = ""

		claims, err := extractDesiredClaims(renderedObjects(t, pvc))

		require.ErrorContains(t, err, "rendered PVC \"data\" has no namespace")
		assert.Empty(t, claims)
	})

	t.Run("rejects a StatefulSet without a namespace", func(t *testing.T) {
		statefulSet := newStatefulSet("postgres", ptr.To[int32](1), 0, newPVC("data", "1Gi", nil))
		statefulSet.Namespace = ""

		claims, err := extractDesiredClaims(renderedObjects(t, statefulSet))

		require.ErrorContains(t, err, "rendered StatefulSet \"postgres\" has no namespace")
		assert.Empty(t, claims)
	})

	t.Run("rejects a rendered PVC without a storage request", func(t *testing.T) {
		pvc := newPVC("data", "1Gi", nil)
		pvc.Spec.Resources.Requests = nil

		claims, err := extractDesiredClaims(renderedObjects(t, pvc))

		require.ErrorContains(t, err, "rendered PVC \"ecosystem/data\" has no storage request")
		assert.Empty(t, claims)
	})

	tests := []struct {
		apiVersion string
		kind       string
	}{
		{apiVersion: "v1", kind: "PersistentVolumeClaim"},
		{apiVersion: "apps/v1", kind: "StatefulSet"},
	}

	for _, tt := range tests {
		t.Run("rejects malformed "+tt.kind, func(t *testing.T) {
			object := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": tt.apiVersion,
				"kind":       tt.kind,
				"metadata": map[string]any{
					"name":      "broken",
					"namespace": testNamespace,
				},
				"spec": "invalid",
			}}

			claims, err := extractDesiredClaims([]*unstructured.Unstructured{object})

			require.ErrorContains(t, err, "failed to convert rendered "+tt.kind)
			assert.Empty(t, claims)
		})
	}

	t.Run("ignores recognized kinds from custom API groups", func(t *testing.T) {
		objects := []*unstructured.Unstructured{
			{Object: map[string]any{"apiVersion": "example.com/v1", "kind": "PersistentVolumeClaim"}},
			{Object: map[string]any{"apiVersion": "example.com/v1", "kind": "StatefulSet"}},
		}

		claims, err := extractDesiredClaims(objects)

		require.NoError(t, err)
		assert.Empty(t, claims)
	})

	t.Run("returns all rendered object errors", func(t *testing.T) {
		pvc := newPVC("data", "1Gi", nil)
		pvc.Namespace = ""
		statefulSet := newStatefulSet("postgres", ptr.To[int32](1), 0, newPVC("data", "1Gi", nil))
		statefulSet.Namespace = ""

		claims, err := extractDesiredClaims(renderedObjects(t, pvc, statefulSet))

		require.ErrorContains(t, err, "rendered PVC \"data\" has no namespace")
		require.ErrorContains(t, err, "rendered StatefulSet \"postgres\" has no namespace")
		assert.Empty(t, claims)
	})
}

func newPVC(name, storage string, storageClass *string) *corev1.PersistentVolumeClaim {
	storageQuantity := resource.MustParse(storage)
	return &corev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: storageClass,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{corev1.ResourceStorage: storageQuantity.DeepCopy()},
			},
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase:    corev1.ClaimBound,
			Capacity: corev1.ResourceList{corev1.ResourceStorage: storageQuantity.DeepCopy()},
		},
	}
}

func newStatefulSet(name string, replicas *int32, startOrdinal int32, templates ...*corev1.PersistentVolumeClaim) *appsv1.StatefulSet {
	volumeClaimTemplates := make([]corev1.PersistentVolumeClaim, 0, len(templates))
	for _, template := range templates {
		volumeClaimTemplates = append(volumeClaimTemplates, *template.DeepCopy())
	}

	statefulSet := &appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testNamespace},
		Spec: appsv1.StatefulSetSpec{
			Replicas:             replicas,
			VolumeClaimTemplates: volumeClaimTemplates,
		},
	}
	if startOrdinal != 0 {
		statefulSet.Spec.Ordinals = &appsv1.StatefulSetOrdinals{Start: startOrdinal}
	}

	return statefulSet
}

func renderedObjects(t *testing.T, objects ...client.Object) []*unstructured.Unstructured {
	t.Helper()

	result := make([]*unstructured.Unstructured, 0, len(objects))
	for _, object := range objects {
		content, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		require.NoError(t, err)
		result = append(result, &unstructured.Unstructured{Object: content})
	}

	return result
}

func resizeRequestsByName(requests []ResizeRequest) map[string]ResizeRequest {
	result := make(map[string]ResizeRequest, len(requests))
	for _, request := range requests {
		result[request.PVC.Name] = request
	}

	return result
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
