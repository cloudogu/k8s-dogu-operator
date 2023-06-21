package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestResourceGenerator_CreateDoguPVC(t *testing.T) {

	t.Run("Return simple pvc", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		actualPVC, err := generator.CreateDoguPVC(ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedDoguPVC(t), actualPVC)
	})

	t.Run("Return simple pvc with custom size", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)
		sizeBefore := ldapDoguResource.Spec.Resources.DataVolumeSize
		defer func() { ldapDoguResource.Spec.Resources.DataVolumeSize = sizeBefore }()
		ldapDoguResource.Spec.Resources.DataVolumeSize = "6Gi"

		// when
		actualPVC, err := generator.CreateDoguPVC(ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Equal(t, readLdapDoguExpectedDoguPVCWithCustomSize(t), actualPVC)
	})

	t.Run("Return error when reference owner cannot be set", func(t *testing.T) {
		// given
		generator := resourceGenerator{
			scheme: getTestScheme(),
		}

		ldapDoguResource := readLdapDoguResource(t)
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()

		// when
		_, err := generator.CreateDoguPVC(ldapDoguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set controller reference:")
	})
}

func Test_createVolumeByClient(t *testing.T) {
	t.Run("should fail due to invalid config map type content", func(t *testing.T) {
		// given
		volumeClient := core.VolumeClient{
			Name: "k8s-dogu-operator",
			Params: volumeParams{
				Type:    "configmap",
				Content: "invalid",
			},
		}
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				volumeClient,
			},
		}

		// when
		_, err := createVolumeByClient(doguVolume, &volumeClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read configmap client type content of volume my-volume")
	})
	t.Run("should fail due to unsupported client param type", func(t *testing.T) {
		// given
		volumeClient := core.VolumeClient{
			Name: "k8s-dogu-operator",
			Params: volumeParams{
				Type: "invalid",
			},
		}
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				volumeClient,
			},
		}

		// when
		_, err := createVolumeByClient(doguVolume, &volumeClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unsupported client param type invalid in volume my-volume")
	})
}
