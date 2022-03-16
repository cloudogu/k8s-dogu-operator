package controllers

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

const nodeMasterFile = "node-master-file"

// ResourceGenerator generate k8s resources for a given dogu. All resources will be referenced with the dogu resource
// as controller
type ResourceGenerator struct {
	scheme *runtime.Scheme
}

// NewResourceGenerator creates a new generator for k8s resources
func NewResourceGenerator(scheme *runtime.Scheme) *ResourceGenerator {
	return &ResourceGenerator{scheme: scheme}
}

// GetDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource
func (r *ResourceGenerator) GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	volumes := getVolumesForDogu(doguResource, dogu)
	volumeMounts := getVolumeMountsForDogu(doguResource, dogu)

	// Create deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.Name,
		Namespace: doguResource.Namespace,
	}}

	labels := map[string]string{"dogu": doguResource.Name}
	deployment.ObjectMeta.Labels = labels
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Strategy: appsv1.DeploymentStrategy{
			Type: "Recreate",
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry-cloudogu-com"}},
				Hostname:         doguResource.Name,
				Volumes:          volumes,
				Containers: []corev1.Container{{
					Name:            doguResource.Name,
					Image:           dogu.Image + ":" + dogu.Version,
					ImagePullPolicy: corev1.PullIfNotPresent,
					VolumeMounts:    volumeMounts}},
			},
		},
	}

	if len(dogu.Volumes) > 0 {
		group, _ := strconv.Atoi(dogu.Volumes[0].Group)
		gid := int64(group)
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			FSGroup: &gid,
		}
	}

	err := ctrl.SetControllerReference(doguResource, deployment, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return deployment, nil
}

func getVolumesForDogu(doguResource *k8sv1.Dogu, dogu *core.Dogu) []corev1.Volume {
	nodeMasterVolume := corev1.Volume{
		Name: nodeMasterFile,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: nodeMasterFile},
			},
		},
	}

	mode := int32(0744)

	privateVolume := corev1.Volume{
		Name: doguResource.GetPrivateVolumeName(),
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  doguResource.GetPrivateVolumeName(),
				DefaultMode: &mode,
			},
		},
	}

	volumes := []corev1.Volume{
		nodeMasterVolume,
		privateVolume,
	}

	if len(dogu.Volumes) > 0 {
		dataVolume := corev1.Volume{
			Name: doguResource.GetDataVolumeName(),
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: doguResource.Name,
				},
			},
		}
		volumes = append(volumes, dataVolume)
	}

	return volumes
}

func getVolumeMountsForDogu(doguResource *k8sv1.Dogu, dogu *core.Dogu) []corev1.VolumeMount {
	doguVolumeMounts := []corev1.VolumeMount{
		{
			Name:      nodeMasterFile,
			ReadOnly:  true,
			MountPath: "/etc/ces/node_master",
			SubPath:   "node_master",
		},
		{
			Name:      doguResource.GetPrivateVolumeName(),
			ReadOnly:  true,
			MountPath: "/private",
		},
	}

	for _, doguVolume := range dogu.Volumes {
		newVolume := corev1.VolumeMount{
			Name:      doguResource.GetDataVolumeName(),
			ReadOnly:  false,
			MountPath: doguVolume.Path,
			SubPath:   doguVolume.Name,
		}
		doguVolumeMounts = append(doguVolumeMounts, newVolume)
	}

	return doguVolumeMounts
}

// GetDoguService creates a new instance of a service with the given dogu custom resource and container image.
// The container image is used to extract the exposed ports
func (r *ResourceGenerator) GetDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.Name,
			Namespace: doguResource.Namespace,
			Labels:    map[string]string{"app": cesLabel, "dogu": doguResource.Name},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"dogu": doguResource.Name},
			Ports:    []corev1.ServicePort{},
		},
	}

	for exposedPort := range imageConfig.Config.ExposedPorts {
		port, protocol, err := splitPortConfig(exposedPort)
		if err != nil {
			return service, fmt.Errorf("error splitting port config: %w", err)
		}
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     strconv.Itoa(int(port)),
			Protocol: protocol,
			Port:     port,
		})
	}

	service.Spec.Selector = map[string]string{"dogu": doguResource.Name}

	err := ctrl.SetControllerReference(doguResource, service, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return service, nil
}

func splitPortConfig(exposedPort string) (int32, corev1.Protocol, error) {
	portAndPotentiallyProtocol := strings.Split(exposedPort, "/")

	port, err := strconv.Atoi(portAndPotentiallyProtocol[0])
	if err != nil {
		return 0, "", fmt.Errorf("error parsing int: %w", err)
	}

	if len(portAndPotentiallyProtocol) == 2 {
		return int32(port), corev1.Protocol(strings.ToUpper(portAndPotentiallyProtocol[1])), nil
	}

	return int32(port), corev1.ProtocolTCP, nil
}

// GetDoguPVC creates a persistentvolumeclaim with a 5Gi storage for the given dogu
func (r *ResourceGenerator) GetDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error) {
	doguPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.Name,
			Namespace: doguResource.Namespace,
		},
	}

	doguPvc.ObjectMeta.Labels = map[string]string{"app": cesLabel, "dogu": doguResource.Name}
	doguPvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}

	err := ctrl.SetControllerReference(doguResource, doguPvc, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return doguPvc, nil
}

// GetDoguSecret generates a secret with a given data map for the dogu
func (r *ResourceGenerator) GetDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.GetPrivateVolumeName(),
		Namespace: doguResource.Namespace,
		Labels:    map[string]string{"app": cesLabel, "dogu": doguResource.Name}},
		StringData: stringData}

	err := ctrl.SetControllerReference(doguResource, secret, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return secret, nil
}

func wrapControllerReferenceError(err error) error {
	return fmt.Errorf("failed to set controller reference: %w", err)
}
