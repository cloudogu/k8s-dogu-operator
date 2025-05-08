package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme, opts ...controllerutil.OwnerReferenceOption) error {
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

	t.Run("should create only one pvc-Volume and no emptyDir-Volumes (will be created in static Volumes)", func(t *testing.T) {
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
		assert.Len(t, volumes, 2)
		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumes[0].Name)
		assert.Equal(t, doguVolumes[3].Name, volumes[1].Name)
	})
}

func Test_createVolumeMounts(t *testing.T) {
	t.Run("should create static volumeMounts", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumeMounts := createVolumeMounts(ldapDoguResource, &core.Dogu{})

		// then
		assert.Len(t, volumeMounts, 6)

		assert.Equal(t, doguHealth, volumeMounts[0].Name)
		assert.True(t, volumeMounts[0].ReadOnly)
		assert.Equal(t, "/etc/ces/health", volumeMounts[0].MountPath)

		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumeMounts[1].Name)
		assert.False(t, volumeMounts[1].ReadOnly)
		assert.Equal(t, "/var/ces/state", volumeMounts[1].MountPath)
		assert.Equal(t, "state", volumeMounts[1].SubPath)

		assert.Equal(t, globalConfig, volumeMounts[2].Name)
		assert.True(t, volumeMounts[2].ReadOnly)
		assert.Equal(t, "/etc/ces/config/global", volumeMounts[2].MountPath)

		assert.Equal(t, normalConfig, volumeMounts[3].Name)
		assert.True(t, volumeMounts[3].ReadOnly)
		assert.Equal(t, "/etc/ces/config/normal", volumeMounts[3].MountPath)

		assert.Equal(t, sensitiveConfig, volumeMounts[4].Name)
		assert.True(t, volumeMounts[4].ReadOnly)
		assert.Equal(t, "/etc/ces/config/sensitive", volumeMounts[4].MountPath)
	})

	t.Run("should create own dogu.json volume mount", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumeMounts := createVolumeMounts(ldapDoguResource, &core.Dogu{Name: "official/ldap"})

		// then
		assert.Len(t, volumeMounts, 6)

		assert.Equal(t, "ldap-dogu-json", volumeMounts[5].Name)
		assert.True(t, volumeMounts[5].ReadOnly)
		assert.Equal(t, "/etc/ces/dogu_json/ldap", volumeMounts[5].MountPath)
	})

	t.Run("should create dogu volumeMounts", func(t *testing.T) {
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
			Name:    "official/ldap",
			Volumes: volumes,
		})

		// then
		assert.Len(t, volumeMounts, 9)

		assert.Equal(t, "ldap-dogu-json", volumeMounts[5].Name)
		assert.True(t, volumeMounts[5].ReadOnly)
		assert.Equal(t, "/etc/ces/dogu_json/ldap", volumeMounts[5].MountPath)

		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumeMounts[6].Name)
		assert.False(t, volumeMounts[6].ReadOnly)
		assert.Equal(t, volumes[0].Path, volumeMounts[6].MountPath)
		assert.Equal(t, volumes[0].Name, volumeMounts[6].SubPath)

		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumeMounts[7].Name)
		assert.False(t, volumeMounts[7].ReadOnly)
		assert.Equal(t, volumes[1].Path, volumeMounts[7].MountPath)
		assert.Equal(t, volumes[1].Name, volumeMounts[7].SubPath)

		assert.Equal(t, volumes[2].Name, volumeMounts[8].Name)
		assert.False(t, volumeMounts[8].ReadOnly)
		assert.Equal(t, volumes[2].Path, volumeMounts[8].MountPath)
	})
}

func Test_createVolumes(t *testing.T) {
	t.Run("should create static volumes", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{}, false)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 6)

		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumes[1].Name)
	})

	t.Run("should create own dogu.json volume", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{Name: "official/ldap"}, false)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 6)

		assert.Equal(t, "ldap-dogu-json", volumes[5].Name)
		assert.Equal(t, "dogu-spec-ldap", volumes[5].VolumeSource.ConfigMap.LocalObjectReference.Name)
		assert.True(t, *volumes[5].VolumeSource.ConfigMap.Optional)
	})

	t.Run("should create importer publicKey-volume for active export-mode", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{}, true)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 7)

		assert.Equal(t, importPublicKeyVolumeName, volumes[6].Name)
		assert.Equal(t, importerPublicKeyConfigMapName, volumes[6].VolumeSource.ConfigMap.Name)
		assert.True(t, *volumes[6].VolumeSource.ConfigMap.Optional)
	})

	t.Run("should create dogu volumes", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)

		// when
		volumes, err := createVolumes(ldapDoguResource, &core.Dogu{
			Name: "official/ldap",
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
		}, false)

		// then
		require.NoError(t, err)
		assert.Len(t, volumes, 7)

		assert.Equal(t, "dogu-health", volumes[0].Name)
		assert.Equal(t, ldapDoguResource.GetEphemeralDataVolumeName(), volumes[1].Name)
		assert.Equal(t, globalConfig, volumes[2].Name)
		assert.Equal(t, normalConfig, volumes[3].Name)
		assert.Equal(t, sensitiveConfig, volumes[4].Name)
		assert.Equal(t, "ldap-dogu-json", volumes[5].Name)
		assert.Equal(t, ldapDoguResource.GetDataVolumeName(), volumes[6].Name)
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
		}, false)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read k8s-dogu-operator client params of volume Vol1")
	})
}

func Test_createDoguJsonVolumesFromDependencies(t *testing.T) {
	optionalTrue := true

	type args struct {
		dogu *core.Dogu
	}
	tests := []struct {
		name string
		args args
		want []corev1.Volume
	}{
		{
			name: "should create dogu json volumes for dependencies and optional dependencies",
			args: args{dogu: readCasDogu(t)},
			want: []corev1.Volume{
				{
					Name: "nginx-dogu-json",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "dogu-spec-nginx",
							},
							Optional: &optionalTrue,
						},
					},
				},
				{
					Name: "postfix-dogu-json",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "dogu-spec-postfix",
							},
							Optional: &optionalTrue,
						},
					},
				},
				{
					Name: "ldap-dogu-json",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "dogu-spec-ldap",
							},
							Optional: &optionalTrue,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, createDoguJsonVolumesFromDependencies(tt.args.dogu), "createDoguJsonVolumesFromDependencies(%v)", tt.args.dogu)
		})
	}
}

func Test_createDoguJsonVolumeMountsFromDependencies(t *testing.T) {
	type args struct {
		dogu *core.Dogu
	}
	tests := []struct {
		name string
		args args
		want []corev1.VolumeMount
	}{
		{
			name: "should create dogu json volume mounts for dependencies and optional dependencies",
			args: args{dogu: readCasDogu(t)},
			want: []corev1.VolumeMount{
				{
					Name:      "nginx-dogu-json",
					ReadOnly:  true,
					MountPath: "/etc/ces/dogu_json/nginx",
				},
				{
					Name:      "postfix-dogu-json",
					ReadOnly:  true,
					MountPath: "/etc/ces/dogu_json/postfix",
				},
				{
					Name:      "ldap-dogu-json",
					ReadOnly:  true,
					MountPath: "/etc/ces/dogu_json/ldap",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, createDoguJsonVolumeMountsFromDependencies(tt.args.dogu), "createDoguJsonVolumeMountsFromDependencies(%v)", tt.args.dogu)
		})
	}
}

func Test_createExporterSidecarVolumeMounts(t *testing.T) {
	t.Run("should create volume-mount for exporter sidecar", func(t *testing.T) {
		mounts := createExporterSidecarVolumeMounts(&k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{})

		require.NotNil(t, mounts)
		require.Len(t, mounts, 1)

		assert.Equal(t, importPublicKeyVolumeName, mounts[0].Name)
		assert.Equal(t, "/root/.ssh/authorized_keys", mounts[0].MountPath)
		assert.Equal(t, importerPublicKeySubPath, mounts[0].SubPath)
		assert.True(t, mounts[0].ReadOnly)
	})

	t.Run("should create volume-mount for exporter sidecar including data-volume", func(t *testing.T) {
		mounts := createExporterSidecarVolumeMounts(&k8sv2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}, &core.Dogu{Volumes: []core.Volume{{NeedsBackup: true}}})

		require.NotNil(t, mounts)
		require.Len(t, mounts, 2)

		assert.Equal(t, importPublicKeyVolumeName, mounts[0].Name)
		assert.Equal(t, "/root/.ssh/authorized_keys", mounts[0].MountPath)
		assert.Equal(t, importerPublicKeySubPath, mounts[0].SubPath)
		assert.True(t, mounts[0].ReadOnly)

		assert.Equal(t, "test-data", mounts[1].Name)
		assert.Equal(t, "/data", mounts[1].MountPath)
		assert.Equal(t, "", mounts[1].SubPath)
	})
}
