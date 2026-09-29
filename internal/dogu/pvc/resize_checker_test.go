package pvc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testNamespace = "ecosystem"

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
