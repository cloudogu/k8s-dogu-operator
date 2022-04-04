package controllers

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
)

const (
	cesLabel       = "ces"
	nodeMasterFile = "node-master-file"
)

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

	fsGroupChangePolicy := corev1.FSGroupChangeOnRootMismatch

	if len(dogu.Volumes) > 0 {
		group, _ := strconv.Atoi(dogu.Volumes[0].Group)
		gid := int64(group)
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			FSGroup:             &gid,
			FSGroupChangePolicy: &fsGroupChangePolicy,
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
		port, protocol, err := annotation.SplitImagePortConfig(exposedPort)
		if err != nil {
			return service, fmt.Errorf("error splitting port config: %w", err)
		}
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:     strconv.Itoa(int(port)),
			Protocol: protocol,
			Port:     port,
		})
	}

	cesServiceAnnotationCreator := annotation.CesServiceAnnotator{}
	err := cesServiceAnnotationCreator.AnnotateService(service, &imageConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to annotate service: %w", err)
	}

	err = ctrl.SetControllerReference(doguResource, service, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return service, nil
}

// GetDoguExposedServices creates a new instance of a LoadBalancer service for each exposed port.
func (r *ResourceGenerator) GetDoguExposedServices(doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]corev1.Service, error) {
	exposedServices := []corev1.Service{}

	for _, exposedPort := range dogu.ExposedPorts {
		ipSingleStackPolicy := corev1.IPFamilyPolicySingleStack
		exposedService := corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-exposed-%d", doguResource.Name, exposedPort.Host),
				Namespace: doguResource.Namespace,
				Labels:    map[string]string{"app": cesLabel, "dogu": doguResource.Name},
			},
			Spec: corev1.ServiceSpec{
				Type:           corev1.ServiceTypeLoadBalancer,
				IPFamilyPolicy: &ipSingleStackPolicy,
				IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
				Selector:       map[string]string{"dogu": doguResource.Name},
				Ports: []corev1.ServicePort{{
					Name:       strconv.Itoa(exposedPort.Host),
					Protocol:   corev1.Protocol(strings.ToUpper(exposedPort.Type)),
					Port:       int32(exposedPort.Host),
					TargetPort: intstr.FromInt(exposedPort.Container),
				}},
			},
		}

		err := ctrl.SetControllerReference(doguResource, &exposedService, r.scheme)
		if err != nil {
			return nil, wrapControllerReferenceError(err)
		}

		exposedServices = append(exposedServices, exposedService)
	}

	return exposedServices, nil
}

// GetDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu
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
