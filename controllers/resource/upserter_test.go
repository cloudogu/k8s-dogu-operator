package resource

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"testing"
)

func TestNewUpserter(t *testing.T) {
}

func Test_longhornPVCValidator_validate(t *testing.T) {
	t.Run("error on validating pvc with non pvc object", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testObject := &appsv1.Deployment{}

		// when
		err := validator.validate(context.Background(), "name", testObject)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported validation object (expected: PVC)")
	})

	t.Run("error on missing beta longhorn annotation", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
		}

		// when
		err := validator.validate(context.Background(), "name", testPvc)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as annotation [volume.beta.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
	})

	t.Run("error on missing default longhorn annotation", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
				},
			},
		}

		// when
		err := validator.validate(context.Background(), "name", testPvc)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as annotation [volume.kubernetes.io/storage-provisioner] does not exist or is not [driver.longhorn.io]")
	})

	t.Run("error on missing dogu label", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
			},
		}

		// when
		err := validator.validate(context.Background(), "name", testPvc)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as pvc does not contain label [dogu] with value [name]")
	})

	t.Run("error on missing dogu label", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
				Labels: map[string]string{"dogu": "name"},
			},
			Spec: v1.PersistentVolumeClaimSpec{StorageClassName: pointer.String("invalidStorageClass")},
		}

		// when
		err := validator.validate(context.Background(), "name", testPvc)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pvc for dogu [name] is not valid as pvc has invalid storage class: the storage class must be [longhorn]")
	})

	t.Run("success", func(t *testing.T) {
		// given
		validator := longhornPVCValidator{}
		testPvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
				Annotations: map[string]string{
					annotationKubernetesBetaVolumeDriver: longhornDiverID,
					annotationKubernetesVolumeDriver:     longhornDiverID,
				},
				Labels: map[string]string{"dogu": "name"},
			},
			Spec: v1.PersistentVolumeClaimSpec{StorageClassName: pointer.String(longhornStorageClassName)},
		}

		// when
		err := validator.validate(context.Background(), "name", testPvc)

		// given
		require.NoError(t, err)
	})
}

func Test_upserter_ApplyDoguResource(t *testing.T) {
}

func Test_upserter_updateOrInsert(t *testing.T) {
}

func Test_upserter_upsertDoguDeployment(t *testing.T) {
}

func Test_upserter_upsertDoguExposedServices(t *testing.T) {
}

func Test_upserter_upsertDoguPVC(t *testing.T) {
}

func Test_upserter_upsertDoguService(t *testing.T) {
}
