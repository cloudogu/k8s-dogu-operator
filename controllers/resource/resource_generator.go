package resource

import (
	"fmt"
	"strconv"
	"strings"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

const (
	appLabelKey      = "app"
	appLabelValueCes = "ces"
)

const (
	nodeMasterFile = "node-master-file"
)

const (
	DoguReservedPath = "/tmp/dogu-reserved"
)

const (
	doguPodNamespace = "POD_NAMESPACE"
	doguPodName      = "POD_NAME"
)

const (
	chownInitContainerName  = "dogu-volume-chown-init"
	chownInitContainerImage = "busybox:1.36"
)

// kubernetesServiceAccountKind describes a service account on kubernetes.
const kubernetesServiceAccountKind = "k8s"

// resourceGenerator generate k8s resources for a given dogu. All resources will be referenced with the dogu resource
// as controller
type resourceGenerator struct {
	scheme           *runtime.Scheme
	doguLimitPatcher cloudogu.LimitPatcher
}

// NewResourceGenerator creates a new generator for k8s resources
func NewResourceGenerator(scheme *runtime.Scheme, limitPatcher cloudogu.LimitPatcher) *resourceGenerator {
	return &resourceGenerator{scheme: scheme, doguLimitPatcher: limitPatcher}
}

// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
func (r *resourceGenerator) CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	podTemplate, err := r.GetPodTemplate(doguResource, dogu)
	if err != nil {
		return nil, err
	}

	// Don't use the dogu.version label in deployment since it cannot be updated in the spec.
	// Version labels only get applied to pods to discern them during an upgrade.
	appDoguNameLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())

	// Create deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.Name,
		Namespace: doguResource.Namespace,
		Labels:    appDoguNameLabels,
	}}

	deployment.Spec = buildDeploymentSpec(doguResource.GetDoguNameLabel(), podTemplate)

	fsGroupChangePolicy := corev1.FSGroupChangeOnRootMismatch

	if len(dogu.Volumes) > 0 {
		group, _ := strconv.Atoi(dogu.Volumes[0].Group)
		gid := int64(group)
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			FSGroup:             &gid,
			FSGroupChangePolicy: &fsGroupChangePolicy,
		}
	}

	doguLimits, err := r.doguLimitPatcher.RetrievePodLimits(doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve resource limits for dogu [%s]: %w", doguResource.Name, err)
	}

	err = r.doguLimitPatcher.PatchDeployment(deployment, doguLimits)
	if err != nil {
		return nil, fmt.Errorf("failed to patch resource limits into dogu deployment [%s]: %w", doguResource.Name, err)
	}

	err = ctrl.SetControllerReference(doguResource, deployment, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return deployment, nil
}

// GetPodTemplate returns a pod template for the given dogu.
func (r *resourceGenerator) GetPodTemplate(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*corev1.PodTemplateSpec, error) {
	volumes, err := createVolumes(doguResource, dogu)
	if err != nil {
		return nil, err
	}

	volumeMounts := createVolumeMounts(doguResource, dogu)
	envVars := []corev1.EnvVar{
		{Name: doguPodNamespace, Value: doguResource.GetNamespace()},
		{Name: doguPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		}}}
	var startupProbe *corev1.Probe
	var livenessProbe *corev1.Probe
	var command []string
	var args []string
	startupProbe = CreateStartupProbe(dogu)
	livenessProbe = createLivenessProbe(dogu)
	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	allLabels := GetAppLabel().Add(doguResource.GetPodLabels())

	// Avoid env vars like <service_name>_PORT="tcp://<ip>:<port>" because they could override regular dogu env vars.
	enableServiceLinks := false

	chownContainer, err := getChownInitContainer(dogu, doguResource)
	if err != nil {
		return nil, err
	}
	var initContainers []corev1.Container
	if chownContainer != nil {
		initContainers = append(initContainers, *chownContainer)
	}

	podTemplate := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: allLabels,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "k8s-dogu-operator-docker-registry"}},
			Hostname:           doguResource.Name,
			Volumes:            volumes,
			EnableServiceLinks: &enableServiceLinks,
			InitContainers:     initContainers,
			Containers: []corev1.Container{{
				Command:         command,
				Args:            args,
				LivenessProbe:   livenessProbe,
				StartupProbe:    startupProbe,
				Name:            doguResource.Name,
				Image:           dogu.Image + ":" + dogu.Version,
				ImagePullPolicy: pullPolicy,
				VolumeMounts:    volumeMounts,
				Env:             envVars,
			}},
		},
	}

	accountName, ok := getKubernetesServiceAccount(dogu)
	if ok {
		podTemplate.Spec.ServiceAccountName = accountName
	}

	return podTemplate, nil
}

func getChownInitContainer(dogu *core.Dogu, doguResource *k8sv1.Dogu) (*corev1.Container, error) {
	// Skip chown volumes with dogu-operator client because these are volumes from configmaps and read only.
	filteredVolumes := filterVolumesWithClient(dogu.Volumes, doguOperatorClient)
	if len(filteredVolumes) == 0 {
		return nil, nil
	}

	var commands []string
	for _, volume := range filteredVolumes {
		uid, err := strconv.ParseInt(volume.Owner, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse owner id %s from volume %s: %w", volume.Owner, volume.Name, err)
		}
		gid, err := strconv.ParseInt(volume.Group, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse group id %s from volume %s: %w", volume.Group, volume.Name, err)
		}

		if uid > 0 && gid > 0 {
			mkdirCommand := fmt.Sprintf("mkdir -p \"%s\"", volume.Path)
			chownCommand := fmt.Sprintf("chown -R %s:%s \"%s\"", volume.Owner, volume.Group, volume.Path)
			commands = append(commands, mkdirCommand)
			commands = append(commands, chownCommand)
		} else {
			return nil, fmt.Errorf("owner %d or group %d are not greater than 0", uid, gid)
		}
	}

	return &corev1.Container{
		Name:         chownInitContainerName,
		Image:        chownInitContainerImage,
		Command:      []string{"sh", "-c", strings.Join(commands, " && ")},
		VolumeMounts: createDoguVolumeMounts(doguResource, dogu),
	}, nil
}

func filterVolumesWithClient(volumes []core.Volume, client string) []core.Volume {
	var filteredList []core.Volume
	for _, volume := range volumes {
		_, clientExists := volume.GetClient(client)
		if clientExists {
			continue
		}
		filteredList = append(filteredList, volume)
	}

	return filteredList
}

func getKubernetesServiceAccount(dogu *core.Dogu) (string, bool) {
	for _, account := range dogu.ServiceAccounts {
		if account.Kind == kubernetesServiceAccountKind && account.Type == doguOperatorClient {
			return dogu.GetSimpleName(), true
		}
	}

	return "", false
}

func buildDeploymentSpec(selectorLabels map[string]string, podTemplate *corev1.PodTemplateSpec) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
		Strategy: appsv1.DeploymentStrategy{
			Type: "Recreate",
		},
		Template: *podTemplate,
	}
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
				// Setting this value to low makes some dogus unable to start that require a certain amount of time.
				// The default value is set to 30 min.
				FailureThreshold: 6 * 30,
			}
		}
	}
	return nil
}

func CreateStartupProbe(dogu *core.Dogu) *corev1.Probe {
	for _, healthCheck := range dogu.HealthChecks {
		if healthCheck.Type == "state" {
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{Command: []string{"bash", "-c", "[[ $(doguctl state) == \"ready\" ]]"}},
				},
				TimeoutSeconds:   1,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				// Setting this value to low makes some dogus unable to start that require a certain amount of time.
				// The default value is set to 30 min.
				FailureThreshold: 6 * 30,
			}
		}
	}
	return nil
}

// GetAppLabel returns an app label which all CES resource may receive for general selection.
func GetAppLabel() k8sv1.CesMatchingLabels {
	return map[string]string{appLabelKey: appLabelValueCes}
}

// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
// the service ports to the created service.
func (r *resourceGenerator) CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error) {
	appDoguLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.Name,
			Namespace: doguResource.Namespace,
			Labels:    appDoguLabels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: doguResource.GetDoguNameLabel(),
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
	appDoguLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())

	for _, exposedPort := range dogu.ExposedPorts {
		ipSingleStackPolicy := corev1.IPFamilyPolicySingleStack
		exposedService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-exposed-%d", doguResource.Name, exposedPort.Host),
				Namespace: doguResource.Namespace,
				Labels:    appDoguLabels,
			},
			Spec: corev1.ServiceSpec{
				Type:           corev1.ServiceTypeLoadBalancer,
				IPFamilyPolicy: &ipSingleStackPolicy,
				IPFamilies:     []corev1.IPFamily{corev1.IPv4Protocol},
				Selector:       doguResource.GetDoguNameLabel(),
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

// CreateDoguSecret generates a secret with a given data map for the dogu
func (r *resourceGenerator) CreateDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	appDoguLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.GetPrivateKeySecretName(),
			Namespace: doguResource.Namespace,
			Labels:    appDoguLabels,
		},
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
