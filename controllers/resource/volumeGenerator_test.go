package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

func Test_createDoguVolumes(t *testing.T) {
	t.Run("should create configMap-Volume for doguVolume with client", func(t *testing.T) {
		// given
		volumeClient := core.VolumeClient{
			Name: "k8s-dogu-operator",
			Params: volumeParams{
				Type: "configmap",
				Content: map[string]string{
					"Name": "testCM",
				},
			},
		}
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				volumeClient,
			},
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createDoguVolumes([]core.Volume{doguVolume}, ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 1)
		assert.Equal(t, doguVolume.Name, volumes[0].Name)
		assert.IsType(t, &corev1.ConfigMapVolumeSource{}, volumes[0].VolumeSource.ConfigMap)
	})

	t.Run("should fail to create configMap-Volume for doguVolume with client without name", func(t *testing.T) {
		// given
		volumeClient := core.VolumeClient{
			Name:   "k8s-dogu-operator",
			Params: "invalid",
		}
		doguVolume := core.Volume{
			Name: "my-volume",
			Clients: []core.VolumeClient{
				volumeClient,
			},
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		_, err := createDoguVolumes([]core.Volume{doguVolume}, ldapDoguResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume my-volume")
	})

	t.Run("should create pvc-Volume for doguVolumes with backup", func(t *testing.T) {
		// given
		doguVolume := core.Volume{
			Name:        "my-volume",
			NeedsBackup: true,
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createDoguVolumes([]core.Volume{doguVolume}, ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 1)
		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumes[0].Name)
		assert.IsType(t, &corev1.PersistentVolumeClaimVolumeSource{}, volumes[0].VolumeSource.PersistentVolumeClaim)
		assert.Equal(t, ldapDoguResource.Name, volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName)
	})

	t.Run("should create emptyDir-Volume for doguVolumes without backup", func(t *testing.T) {
		// given
		doguVolume := core.Volume{
			Name:        "my-volume",
			NeedsBackup: false,
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createDoguVolumes([]core.Volume{doguVolume}, ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 1)
		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumes[0].Name)
		assert.IsType(t, &corev1.EmptyDirVolumeSource{}, volumes[0].VolumeSource.EmptyDir)
	})

	t.Run("should create only one pvc-Volume and onyl one emptyDir-Volume", func(t *testing.T) {
		// given
		volumeClient := core.VolumeClient{
			Name: "k8s-dogu-operator",
			Params: volumeParams{
				Type: "configmap",
				Content: map[string]string{
					"Name": "testCM",
				},
			},
		}

		doguVolumes := []core.Volume{
			{
				Name:        "data 1",
				NeedsBackup: true,
			},
			{
				Name:        "ephemeral 1",
				NeedsBackup: false,
			},
			{
				Name:        "data 2",
				NeedsBackup: true,
			},
			{
				Name: "with client",
				Clients: []core.VolumeClient{
					volumeClient,
				},
			},
			{
				Name:        "ephemeral 1",
				NeedsBackup: false,
			},
			{
				Name:        "data 3",
				NeedsBackup: true,
			},
			{
				Name:        "ephemeral 3",
				NeedsBackup: false,
			},
		}

		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createDoguVolumes(doguVolumes, ldapDoguResource)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 3)
		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumes[0].Name)
		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumes[1].Name)
		assert.Equal(t, doguVolumes[3].Name, volumes[2].Name)
	})
}

func Test_createVolumeMounts(t *testing.T) {
	t.Run("should create create static volumeMounts", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumeMounts := createVolumeMounts(ldapDoguResource, &core.Dogu{})

		// then
		assert.Len(t, volumeMounts, 2)

		assert.Equal(t, nodeMasterFile, volumeMounts[0].Name)
		assert.True(t, volumeMounts[0].ReadOnly)
		assert.Equal(t, "/etc/ces/node_master", volumeMounts[0].MountPath)
		assert.Equal(t, "node_master", volumeMounts[0].SubPath)

		assert.Equal(t, ldapDoguResource.GetPrivateKeySecretName(), volumeMounts[1].Name)
		assert.True(t, volumeMounts[1].ReadOnly)
		assert.Equal(t, "/private", volumeMounts[1].MountPath)
	})

	t.Run("should create create dogu volumeMounts", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		volumes := []core.Volume{
			{
				Name:        "Vol1",
				Path:        "/path/one",
				NeedsBackup: true,
				Clients:     nil,
			},
			{
				Name:        "Vol2",
				Path:        "/path/two",
				NeedsBackup: false,
				Clients:     nil,
			},
			{
				Name:        "Vol1",
				Path:        "/path/three",
				NeedsBackup: true,
				Clients:     []core.VolumeClient{{Name: "k8s-dogu-operator"}},
			},
		}

		// when
		volumeMounts := createVolumeMounts(ldapDoguResource, &core.Dogu{
			Volumes: volumes,
		})

		// then
		assert.Len(t, volumeMounts, 5)

		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumeMounts[2].Name)
		assert.False(t, volumeMounts[2].ReadOnly)
		assert.Equal(t, volumes[0].Path, volumeMounts[2].MountPath)
		assert.Equal(t, volumes[0].Name, volumeMounts[2].SubPath)

		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumeMounts[3].Name)
		assert.False(t, volumeMounts[3].ReadOnly)
		assert.Equal(t, volumes[1].Path, volumeMounts[3].MountPath)
		assert.Equal(t, volumes[1].Name, volumeMounts[3].SubPath)

		assert.Equal(t, volumes[2].Name, volumeMounts[4].Name)
		assert.False(t, volumeMounts[4].ReadOnly)
		assert.Equal(t, volumes[2].Path, volumeMounts[4].MountPath)
	})
}

func Test_createVolumes(t *testing.T) {
	t.Run("should create create static volumes", func(t *testing.T) {
		// given
		mode := int32(0744)
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{})

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 2)

		assert.Equal(t, nodeMasterFile, volumes[0].Name)
		assert.Equal(t, nodeMasterFile, volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name)

		assert.Equal(t, ldapDoguResource.GetPrivateKeySecretName(), volumes[1].Name)
		assert.Equal(t, ldapDoguResource.GetPrivateKeySecretName(), volumes[1].VolumeSource.Secret.SecretName)
		assert.Equal(t, &mode, volumes[1].VolumeSource.Secret.DefaultMode)
	})

	t.Run("should create create dogu volumes", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{
			Volumes: []core.Volume{
				{
					Name:        "Vol1",
					Path:        "/path/one",
					NeedsBackup: true,
				},
				{
					Name:        "Vol2",
					Path:        "/path/twi",
					NeedsBackup: false,
				},
			},
		})

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 4)

		assert.Equal(t, nodeMasterFile, volumes[0].Name)
		assert.Equal(t, ldapDoguResource.GetPrivateKeySecretName(), volumes[1].Name)
		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumes[2].Name)
		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumes[3].Name)
	})

	t.Run("should fail create dogu volumes with invalid client-params", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		_, err := createVolumes(ldapDoguResource, &core.Dogu{
			Volumes: []core.Volume{
				{
					Name:        "Vol1",
					Path:        "/path/one",
					NeedsBackup: true,
					Clients: []core.VolumeClient{{
						Name:   "k8s-dogu-operator",
						Params: "invalid",
					}},
				},
			},
		})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume Vol1")
	})
}
