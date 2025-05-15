package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// doguOperatorClient references the k8s-dogu-operator as a client for the creation of a resource.
const doguOperatorClient = "k8s-dogu-operator"

// configMapParamType describes a volume of type config map.
const configMapParamType volumeParamsType = "configmap"

const (
	fmtDoguJsonVolumeName = "%s-dogu-json"
	doguDependencyType    = "dogu"
)

const (
	importPublicKeyVolumeName      = "ces-importer-public-key-volume"
	importerPublicKeyConfigMapName = "ces-importer-public-key"
	importerPublicKeySubPath       = "publicKey"
)

// const names for configs to be mounted
const (
	globalConfig    = "global-config"
	normalConfig    = "normal-config"
	sensitiveConfig = "sensitive-config"
)

// volumeParamsType describes the kind of volume the k8s-dogu-operator should create.
type volumeParamsType string

// volumeParams contains additional information for the k8s-dogu-operator to create a volume.
type volumeParams struct {
	// Type describes the kind of volume the k8s-dogu-operator should create.
	Type volumeParamsType
	// Content contains the actual information that is needed to create a volume of a given Type.
	// The structure of this information is therefore dependent on the Type.
	// To describe a configmap, it could f.i. contain data of type volumeConfigMapContent.
	Content interface{}
}

// volumeConfigMapContent contains information needed to create a volume of type configmap.
type volumeConfigMapContent struct {
	// Name of the configmap to create.
	Name string
}

func CreateVolumes(doguResource *k8sv2.Dogu, dogu *core.Dogu, exportModeActive bool) ([]corev1.Volume, error) {
	volumes := createStaticVolumes(doguResource)
	volumes = append(volumes, createDoguJsonVolumesFromDependencies(dogu)...)
	volumes = append(volumes, getDoguJsonVolumeForDogu(dogu.GetSimpleName()))

	if exportModeActive {
		volumes = append(volumes, createImporterPublicKeyVolume())
	}

	volumesFromDogu, err := createDoguVolumes(dogu.Volumes, doguResource)
	if err != nil {
		return nil, err
	}
	volumes = append(volumes, volumesFromDogu...)

	dataMountVolumes, err := createAdditionalDataVolumes(doguResource)
	if err != nil {
		return nil, err
	}
	volumes = append(volumes, dataMountVolumes...)

	return volumes, nil
}

func createAdditionalDataVolumes(doguResource *k8sv2.Dogu) ([]corev1.Volume, error) {
	// If there is are data mounts with e.g. the same config map only one volume is required.
	var dataMountsByName = map[string]k8sv2.DataMount{}
	var volumes []corev1.Volume
	var multiErr []error
	for _, dataMount := range doguResource.Spec.AdditionalMounts {
		_, ok := dataMountsByName[dataMount.Name]
		if ok {
			continue
		}
		dataMountsByName[dataMount.Name] = dataMount

		mount, err := getVolumeForDataMount(dataMount)
		if err != nil {
			multiErr = append(multiErr, err)
		}
		volumes = append(volumes, mount)
	}

	return volumes, errors.Join(multiErr...)
}

func getVolumeForDataMount(mount k8sv2.DataMount) (corev1.Volume, error) {
	volumeSource := corev1.VolumeSource{}
	// TODO discuss generic usage of volumesource? If yes createAdditionalDataVolumes should be able to create multiple volumes for same source.
	// TODO Add optional flag to CRD?
	switch mount.SourceType {
	case k8sv2.DataSourceConfigMap:
		volumeSource.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mount.Name,
			},
		}
	case k8sv2.DataSourceSecret:
		volumeSource.Secret = &corev1.SecretVolumeSource{
			SecretName: mount.Name,
		}
	default:
		return corev1.Volume{}, fmt.Errorf("volume source %s not supported", mount.SourceType)
	}

	return corev1.Volume{
		Name:         mount.Name,
		VolumeSource: volumeSource,
	}, nil
}

func createDoguJsonVolumesFromDependencies(dogu *core.Dogu) []corev1.Volume {
	var volumes []corev1.Volume
	for _, dependency := range dogu.Dependencies {
		if dependency.Type == doguDependencyType {
			volumes = append(volumes, getDoguJsonVolumeForDogu(dependency.Name))
		}
	}
	for _, dependency := range dogu.OptionalDependencies {
		if dependency.Type == doguDependencyType {
			volumes = append(volumes, getDoguJsonVolumeForDogu(dependency.Name))
		}
	}

	return volumes
}

func getDoguJsonVolumeForDogu(simpleDoguName string) corev1.Volume {
	optional := true
	return corev1.Volume{
		Name: fmt.Sprintf(fmtDoguJsonVolumeName, simpleDoguName),
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("dogu-spec-%s", simpleDoguName),
				},
				Optional: &optional,
			},
		},
	}
}

func createStaticVolumes(doguResource *k8sv2.Dogu) []corev1.Volume {
	doguHealthVolume := corev1.Volume{
		Name: doguHealth,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: doguHealthConfigMap},
			},
		},
	}

	// add EmptyDir-VolumeSource for all dogus to at least give them the ability to write state
	ephemeralVolume := corev1.Volume{
		Name: doguResource.GetEphemeralDataVolumeName(),
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	globalConfigVolume := corev1.Volume{
		Name: globalConfig,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: globalConfig},
			},
		},
	}

	doguConfigName := fmt.Sprintf("%s-config", doguResource.Name)

	normalConfigVolume := corev1.Volume{
		Name: normalConfig,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: doguConfigName},
			},
		},
	}

	sensitiveConfigVolume := corev1.Volume{
		Name: sensitiveConfig,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: doguConfigName,
			},
		},
	}

	return []corev1.Volume{
		doguHealthVolume,
		ephemeralVolume,
		globalConfigVolume,
		normalConfigVolume,
		sensitiveConfigVolume,
	}
}

func createDoguVolumes(doguVolumes []core.Volume, doguResource *k8sv2.Dogu) ([]corev1.Volume, error) {
	var multiError error
	var volumes []corev1.Volume

	// only create max one pvcVolume and one emptyDirVolume
	pvcVolumeCreated := false

	for _, doguVolume := range doguVolumes {
		// to mount e.g. config maps
		client, clientExists := doguVolume.GetClient(doguOperatorClient)
		if clientExists {
			volume, err := createVolumeByClient(doguVolume, client)
			if err != nil {
				multiError = errors.Join(multiError, err)
				continue
			}

			volumes = append(volumes, *volume)
		} else if doguVolume.NeedsBackup && !pvcVolumeCreated {
			// add PVC-VolumeSource for volumes with backup
			dataVolume := corev1.Volume{
				Name: doguResource.GetDataVolumeName(),
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: doguResource.Name,
					},
				},
			}
			volumes = append(volumes, dataVolume)
			pvcVolumeCreated = true
		}
	}

	return volumes, multiError
}

func createVolumeByClient(doguVolume core.Volume, client *core.VolumeClient) (*corev1.Volume, error) {
	clientParams := new(volumeParams)
	err := convertGenericJsonObject(client.Params, clientParams)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s client params of volume %s: %w", doguOperatorClient, doguVolume.Name, err)
	}

	switch clientParams.Type {
	case configMapParamType:
		configMapParamContent := new(volumeConfigMapContent)
		err = convertGenericJsonObject(clientParams.Content, configMapParamContent)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s client type content of volume %s: %w", configMapParamType, doguVolume.Name, err)
		}

		return &corev1.Volume{
			Name: doguVolume.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapParamContent.Name},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported client param type %s in volume %s", clientParams.Type, doguVolume.Name)
	}
}

// convertGenericJsonObject is necessary because go unmarshalls generic json objects as `map[string]interface{}`,
// and, therefore, a type assertion is not possible. This method marshals the generic object (`map[string]interface{}`)
// back into a string. This string is then unmarshalled back into a specific given struct.
func convertGenericJsonObject(genericObject interface{}, targetObject interface{}) error {
	marshalledContent, err := json.Marshal(genericObject)
	if err != nil {
		return err
	}

	err = json.Unmarshal(marshalledContent, targetObject)
	if err != nil {
		return err
	}

	return nil
}

func createVolumeMounts(doguResource *k8sv2.Dogu, dogu *core.Dogu) []corev1.VolumeMount {
	volumeMounts := createStaticVolumeMounts(doguResource)

	// mount dogu jsons from dependency dogus so that a dogu can query attributes from other dogus.
	volumeMounts = append(volumeMounts, createDoguJsonVolumeMountsFromDependencies(dogu)...)
	volumeMounts = append(volumeMounts, getDoguJsonVolumeMountForDogu(dogu.GetSimpleName()))

	return append(volumeMounts, createDoguVolumeMounts(doguResource, dogu)...)
}

func createStaticVolumeMounts(doguResource *k8sv2.Dogu) []corev1.VolumeMount {
	doguVolumeMounts := []corev1.VolumeMount{
		{
			Name:      doguHealth,
			ReadOnly:  true,
			MountPath: "/etc/ces/health",
		},
		{
			Name:      doguResource.GetEphemeralDataVolumeName(),
			ReadOnly:  false,
			MountPath: "/var/ces/state",
			SubPath:   "state",
		},
		{
			Name:      globalConfig,
			ReadOnly:  true,
			MountPath: "/etc/ces/config/global",
		},
		{
			Name:      normalConfig,
			ReadOnly:  true,
			MountPath: "/etc/ces/config/normal",
		},
		{
			Name:      sensitiveConfig,
			ReadOnly:  true,
			MountPath: "/etc/ces/config/sensitive",
		},
	}
	return doguVolumeMounts
}

func createDoguJsonVolumeMountsFromDependencies(dogu *core.Dogu) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	for _, dependency := range dogu.Dependencies {
		if dependency.Type == doguDependencyType {
			volumeMounts = append(volumeMounts, getDoguJsonVolumeMountForDogu(dependency.Name))
		}
	}
	for _, dependency := range dogu.OptionalDependencies {
		if dependency.Type == doguDependencyType {
			volumeMounts = append(volumeMounts, getDoguJsonVolumeMountForDogu(dependency.Name))
		}
	}

	return volumeMounts
}

func getDoguJsonVolumeMountForDogu(simpleDoguName string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      fmt.Sprintf(fmtDoguJsonVolumeName, simpleDoguName),
		ReadOnly:  true,
		MountPath: fmt.Sprintf("/etc/ces/dogu_json/%s", simpleDoguName),
	}
}

func createDoguVolumeMounts(doguResource *k8sv2.Dogu, dogu *core.Dogu) []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	for _, doguVolume := range dogu.Volumes {
		newVolume := createDoguVolumeMount(doguVolume, doguResource)
		volumeMounts = append(volumeMounts, newVolume)
	}

	return volumeMounts
}

func createDoguVolumeMount(doguVolume core.Volume, doguResource *k8sv2.Dogu) corev1.VolumeMount {
	_, clientExists := doguVolume.GetClient(doguOperatorClient)
	if clientExists {
		return corev1.VolumeMount{
			Name:      doguVolume.Name,
			ReadOnly:  false,
			MountPath: doguVolume.Path,
		}
	}

	if !doguVolume.NeedsBackup {
		return corev1.VolumeMount{
			Name:      doguResource.GetEphemeralDataVolumeName(),
			ReadOnly:  false,
			MountPath: doguVolume.Path,
			SubPath:   doguVolume.Name,
		}
	}

	return corev1.VolumeMount{
		Name:      doguResource.GetDataVolumeName(),
		ReadOnly:  false,
		MountPath: doguVolume.Path,
		SubPath:   doguVolume.Name,
	}
}

// CreateDoguPVC creates a persistent volume claim for the given dogu.
func (r *resourceGenerator) CreateDoguPVC(doguResource *k8sv2.Dogu) (*corev1.PersistentVolumeClaim, error) {
	return r.createPVC(doguResource.Name, doguResource, doguResource.GetDataVolumeSize())
}

func (r *resourceGenerator) createPVC(pvcName string, doguResource *k8sv2.Dogu, size resource.Quantity) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: doguResource.Namespace,
			Labels:    GetAppLabel().Add(doguResource.GetDoguNameLabel()),
		},
	}

	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: size,
			},
		},
	}

	err := ctrl.SetControllerReference(doguResource, pvc, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return pvc, nil
}

func createImporterPublicKeyVolume() corev1.Volume {
	optional := true
	return corev1.Volume{
		Name: importPublicKeyVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: importerPublicKeyConfigMapName,
				},
				Optional: &optional,
			},
		},
	}
}

func createExporterSidecarVolumeMounts(doguResource *k8sv2.Dogu, dogu *core.Dogu) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      importPublicKeyVolumeName,
			MountPath: "/root/.ssh/authorized_keys",
			SubPath:   importerPublicKeySubPath,
			ReadOnly:  true,
		},
	}

	if doguHasVolumesWithBackup(dogu) {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      doguResource.GetDataVolumeName(),
			MountPath: "/data",
		})
	}

	return volumeMounts
}

func doguHasVolumesWithBackup(dogu *core.Dogu) bool {
	for _, volume := range dogu.Volumes {
		if volume.NeedsBackup {
			return true
		}
	}

	return false
}
