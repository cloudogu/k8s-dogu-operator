package pvc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDesiredSizes(t *testing.T) {
	t.Run("returns the rendered size of standalone and StatefulSet PVCs", func(t *testing.T) {
		rendered := renderedObjects(t,
			newPVC("data", "2Gi", nil),
			newStatefulSet("postgres", ptr.To[int32](1), 0, newPVC("db", "3Gi", nil)),
		)
		live := []corev1.PersistentVolumeClaim{
			*newPVC("data", "1Gi", nil),
			*newPVC("db-postgres-0", "1Gi", nil),
			*newPVC("db-postgres-1", "1Gi", nil),
		}

		sizes, err := DesiredSizes(rendered, testNamespace, live)

		require.NoError(t, err)
		assert.Equal(t, map[client.ObjectKey]resource.Quantity{
			{Namespace: testNamespace, Name: "data"}:          resource.MustParse("2Gi"),
			{Namespace: testNamespace, Name: "db-postgres-0"}: resource.MustParse("3Gi"),
			{Namespace: testNamespace, Name: "db-postgres-1"}: resource.MustParse("3Gi"),
		}, sizes)
	})

	t.Run("omits live PVCs the chart does not render", func(t *testing.T) {
		rendered := renderedObjects(t,
			newPVC("data", "2Gi", nil),
			newStatefulSet("postgres", ptr.To[int32](1), 0, newPVC("db", "3Gi", nil)),
		)
		live := []corev1.PersistentVolumeClaim{
			*newPVC("existing", "1Gi", nil),
			*newPVC("db-other-0", "1Gi", nil),
		}

		sizes, err := DesiredSizes(rendered, testNamespace, live)

		require.NoError(t, err)
		assert.Empty(t, sizes)
	})

	t.Run("matches rendered objects without a namespace in the target namespace", func(t *testing.T) {
		renderedPVC := newPVC("data", "2Gi", nil)
		renderedPVC.Namespace = ""
		livePVC := newPVC("data", "1Gi", nil)
		livePVC.Namespace = "target"

		sizes, err := DesiredSizes(renderedObjects(t, renderedPVC), "target", []corev1.PersistentVolumeClaim{*livePVC})

		require.NoError(t, err)
		assert.Equal(t, map[client.ObjectKey]resource.Quantity{
			{Namespace: "target", Name: "data"}: resource.MustParse("2Gi"),
		}, sizes)
	})

	t.Run("returns an error for a rendered PVC without storage request", func(t *testing.T) {
		renderedPVC := newPVC("data", "2Gi", nil)
		renderedPVC.Spec.Resources.Requests = nil

		_, err := DesiredSizes(renderedObjects(t, renderedPVC), testNamespace, []corev1.PersistentVolumeClaim{*newPVC("data", "1Gi", nil)})

		require.ErrorContains(t, err, `rendered PVC "ecosystem/data" has no storage request`)
	})
}
