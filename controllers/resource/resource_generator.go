package resource

import (
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	cesLabel       = "ces"
	nodeMasterFile = "node-master-file"
)

const doguReservedPath = "/tmp/dogu-reserved"

const doguPodNamespace = "POD_NAMESPACE"
const doguPodName = "POD_NAME"

// resourceGenerator generate k8s resources for a given dogu. All resources will be referenced with the dogu resource
// as controller
type resourceGenerator struct {
	scheme           *runtime.Scheme
	doguLimitPatcher limitPatcher
}

// NewResourceGenerator creates a new generator for k8s resources
func NewResourceGenerator(scheme *runtime.Scheme, limitPatcher limitPatcher) *resourceGenerator {
	return &resourceGenerator{scheme: scheme, doguLimitPatcher: limitPatcher}
}

type limitPatcher interface {
	// RetrievePodLimits reads all container keys from the dogu configuration and creates a DoguLimits object.
	RetrievePodLimits(doguResource *k8sv1.Dogu) (limit.DoguLimits, error)
	// PatchDeployment patches the given deployment with the resource limits provided.
	PatchDeployment(deployment *appsv1.Deployment, limits limit.DoguLimits) error
}

// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource
func (r *resourceGenerator) CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	volumes := createVolumesForDogu(doguResource, dogu)
	volumeMounts := createVolumeMountsForDogu(doguResource, dogu)
	startupProbe := createStartupProbe(dogu)
	livenessProbe := createLivenessProbe(dogu)

	// Create deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.Name,
		Namespace: doguResource.Namespace,
	}}

	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	labels := map[string]string{"dogu": doguResource.Name}
	deployment.ObjectMeta.Labels = labels

	deployment.Spec = buildDeploymentSpec(doguResource, dogu, labels, volumes, livenessProbe, startupProbe, pullPolicy, volumeMounts)

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

	applyValuesFromCustomDeployment(deployment, customDeployment)

	doguLimits, err := r.doguLimitPatcher.RetrievePodLimits(doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resource limits for dogu [%s]: %w", doguResource.Name, err)
	}

	err = r.doguLimitPatcher.PatchDeployment(deployment, doguLimits)
	if err != nil {
		return nil, fmt.Errorf("failed to patch resource limits into dogu deployment [%s]: %w", doguResource.Name, err)
	}

	return deployment, nil
}

func buildDeploymentSpec(
	doguResource *k8sv1.Dogu,
	dogu *core.Dogu,
	labels map[string]string,
	volumes []corev1.Volume,
	livenessProbe *corev1.Probe,
	startupProbe *corev1.Probe,
	pullPolicy corev1.PullPolicy,
	volumeMounts []corev1.VolumeMount,
) appsv1.DeploymentSpec {
	image := dogu.Image + ":" + dogu.Version

	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Strategy: appsv1.DeploymentStrategy{
			Type: "Recreate",
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				ImagePullSecrets: []corev1.LocalObjectReference{{Name: "k8s-dogu-operator-docker-registry"}},
				Hostname:         doguResource.Name,
				Volumes:          volumes,
				Containers: []corev1.Container{{
					LivenessProbe:   livenessProbe,
					StartupProbe:    startupProbe,
					Name:            doguResource.Name,
					Image:           image,
					ImagePullPolicy: pullPolicy,
					VolumeMounts:    volumeMounts,
					Env: []corev1.EnvVar{
						{Name: doguPodNamespace, Value: doguResource.GetNamespace()},
						{Name: doguPodName, ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						}},
					},
				}},
			},
		},
	}
}

func applyValuesFromCustomDeployment(desiredDeployment *appsv1.Deployment, patchingDeployment *appsv1.Deployment) {
	logger := ctrl.Log

	if patchingDeployment == nil {
		return
	}

	if patchingDeployment.Spec.Template.Spec.ServiceAccountName != "" {
		logger.Info("Found service account in custom deployment from k8s folder. Injecting into the deployment generated by the operator.")
		desiredDeployment.Spec.Template.Spec.ServiceAccountName = patchingDeployment.Spec.Template.Spec.ServiceAccountName
	}

	updateCustomVolumes(desiredDeployment, patchingDeployment)
	updateCustomVolumeMounts(desiredDeployment, patchingDeployment)
}

func updateCustomVolumeMounts(desiredDeployment *appsv1.Deployment, patchingDeployment *appsv1.Deployment) {
	if patchingDeployment.Spec.Template.Spec.Volumes != nil && len(patchingDeployment.Spec.Template.Spec.Volumes) > 0 {
		ctrl.Log.Info("Found custom volumes in custom deployment from k8s folder. Injecting into the deployment generated by the operator.")
		if desiredDeployment.Spec.Template.Spec.Volumes == nil {
			desiredDeployment.Spec.Template.Spec.Volumes = []corev1.Volume{}
		}

		desiredDeployment.Spec.Template.Spec.Volumes = append(desiredDeployment.Spec.Template.Spec.Volumes, patchingDeployment.Spec.Template.Spec.Volumes...)
	}
}

func updateCustomVolumes(desiredDeployment, patchingDeployment *appsv1.Deployment) {
	for i, containerGenerated := range desiredDeployment.Spec.Template.Spec.Containers {
		for _, containerProvided := range patchingDeployment.Spec.Template.Spec.Containers {
			if isContainerToReceiveVolumeMounts(containerGenerated, containerProvided) {
				log.Log.Info("Found custom volume mounts in custom deployment from k8s folder. Injecting into the deployment generated by the operator.")
				if containerGenerated.VolumeMounts == nil {
					containerGenerated.VolumeMounts = []corev1.VolumeMount{}
				}

				containerGenerated.VolumeMounts = append(containerGenerated.VolumeMounts, containerProvided.VolumeMounts...)
				desiredDeployment.Spec.Template.Spec.Containers[i].VolumeMounts = containerGenerated.VolumeMounts
			}
		}
	}
}

func isContainerToReceiveVolumeMounts(containerGenerated, containerProvided corev1.Container) bool {
	return containerGenerated.Name == containerProvided.Name &&
		containerProvided.VolumeMounts != nil && len(containerProvided.VolumeMounts) > 0
}

func createLivenessProbe(dogu *core.Dogu) *corev1.Probe {
	for _, healthCheck := range dogu.HealthChecks {
		if healthCheck.Type == "tcp" {
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					TCPSocket: &corev1.TCPSocketAction{Port: intstr.IntOrString{IntVal: int32(healthCheck.Port)}},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}
		}
	}
	return nil
}

func createStartupProbe(dogu *core.Dogu) *corev1.Probe {
	for _, healthCheck := range dogu.HealthChecks {
		if healthCheck.Type == "state" {
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{Command: []string{"bash", "-c", "[[ $(doguctl state) == \"ready\" ]]"}},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			}
		}
	}
	return nil
}

func createVolumesForDogu(doguResource *k8sv1.Dogu, dogu *core.Dogu) []corev1.Volume {
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

func createVolumeMountsForDogu(doguResource *k8sv1.Dogu, dogu *core.Dogu) []corev1.VolumeMount {
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
		{
			Name:      "dogu-reserved",
			ReadOnly:  false,
			MountPath: doguReservedPath,
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

// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
// the service ports to the created service.
func (r *resourceGenerator) CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error) {
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

// CreateDoguExposedServices creates a new instance of a LoadBalancer service for each exposed port.
// The created service is rather meant for cluster-external access. The given dogu provides the service ports to the
// created service. An additional ingress rule must be created in order to map the arbitrary port to something useful
// (see K8s-service-discovery).
func (r *resourceGenerator) CreateDoguExposedServices(doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]*corev1.Service, error) {
	exposedServices := make([]*corev1.Service, 0)

	for _, exposedPort := range dogu.ExposedPorts {
		ipSingleStackPolicy := corev1.IPFamilyPolicySingleStack
		exposedService := &corev1.Service{
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

		err := ctrl.SetControllerReference(doguResource, exposedService, r.scheme)
		if err != nil {
			return nil, wrapControllerReferenceError(err)
		}

		exposedServices = append(exposedServices, exposedService)
	}

	return exposedServices, nil
}

// CreateDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu
func (r *resourceGenerator) CreateDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error) {
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

// CreateDoguSecret generates a secret with a given data map for the dogu
func (r *resourceGenerator) CreateDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
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
