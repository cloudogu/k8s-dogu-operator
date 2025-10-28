package resource

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

const ReplicaCountStarted = 1
const ReplicaCountStopped = 0

const (
	appLabelKey      = "app"
	appLabelValueCes = "ces"
)

const (
	doguHealthConfigMap = "k8s-dogu-operator-dogu-health"
	doguHealth          = "dogu-health"
)

const (
	doguPodNamespace = "POD_NAMESPACE"
	doguPodName      = "POD_NAME"
	doguPodMultiNode = "ECOSYSTEM_MULTINODE"
)

const (
	chownInitContainerName            = "dogu-volume-chown-init"
	additionalMountsInitContainerName = "dogu-additional-mounts-init"
)

// kubernetesServiceAccountKind describes a service account on kubernetes.
const kubernetesServiceAccountKind = "k8s"

const (
	startupProbeTimoutEnv      = "DOGU_STARTUP_PROBE_TIMEOUT"
	defaultStartupProbeTimeout = 1
)

var (
	additionalMountsDoguMountDir  = fmt.Sprintf("%sdogumount", string(os.PathSeparator))
	additionalMouuntsDataMountDir = fmt.Sprintf("%sdatamount", string(os.PathSeparator))
)

const (
	additionalMountsArg = "copy"
)

// resourceGenerator generate k8s resources for a given dogu. All resources will be referenced with the dogu resource
// as controller
type resourceGenerator struct {
	scheme                   *runtime.Scheme
	requirementsGenerator    RequirementsGenerator
	hostAliasGenerator       HostAliasGenerator
	securityContextGenerator SecurityContextGenerator
	additionalImages         AdditionalImages
}

type AdditionalImages map[string]string

// NewResourceGenerator creates a new generator for k8s resources
func NewResourceGenerator(
	scheme *runtime.Scheme,
	requirementsGenerator RequirementsGenerator,
	hostAliasGenerator HostAliasGenerator,
	securityContextGenerator SecurityContextGenerator,
	additionalImages AdditionalImages,
) DoguResourceGenerator {
	return &resourceGenerator{
		scheme:                   scheme,
		requirementsGenerator:    requirementsGenerator,
		hostAliasGenerator:       hostAliasGenerator,
		securityContextGenerator: securityContextGenerator,
		additionalImages:         additionalImages,
	}
}

func (r *resourceGenerator) UpdateDoguDeployment(ctx context.Context, deployment *appsv1.Deployment, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	return r.updateDoguDeployment(ctx, deployment, doguResource, dogu)
}

// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
func (r *resourceGenerator) CreateDoguDeployment(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	return r.updateDoguDeployment(ctx, &appsv1.Deployment{}, doguResource, dogu)
}

func (r *resourceGenerator) updateDoguDeployment(ctx context.Context, deployment *appsv1.Deployment, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error) {
	podTemplate, err := r.GetPodTemplate(ctx, doguResource, dogu)
	if err != nil {
		return nil, err
	}

	// Don't use the dogu.version label in deployment since it cannot be updated in the spec.
	// Version labels only get applied to pods to discern them during an upgrade.
	appDoguNameLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())

	deployment.Name = doguResource.Name
	deployment.Namespace = doguResource.Namespace
	deployment.Labels = appDoguNameLabels

	updateDeploymentSpec(deployment, doguResource, podTemplate)

	err = ctrl.SetControllerReference(doguResource, deployment, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return deployment, nil
}

// GetPodTemplate returns a pod template for the given dogu.
func (r *resourceGenerator) GetPodTemplate(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (*corev1.PodTemplateSpec, error) {
	volumes, err := CreateVolumes(doguResource, dogu, doguResource.Spec.ExportMode)
	if err != nil {
		return nil, err
	}

	envVars := []corev1.EnvVar{
		{Name: doguPodNamespace, Value: doguResource.GetNamespace()},
		{Name: doguPodName, ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		}},
		{Name: doguPodMultiNode, Value: "true"},
	}

	resourceRequirements, err := r.requirementsGenerator.Generate(ctx, dogu)
	if err != nil {
		return nil, err
	}

	initContainers, err := r.generateInitContainers(ctx, doguResource, dogu, resourceRequirements)
	if err != nil {
		return nil, err
	}

	hostAliases, err := r.hostAliasGenerator.Generate(ctx)
	if err != nil {
		return nil, err
	}

	podTemplate := newPodSpecBuilder(doguResource, dogu).
		labels(GetAppLabel().Add(doguResource.GetPodLabels())).
		annotations(map[string]string{"kubectl.kubernetes.io/default-container": doguResource.Name}).
		hostAliases(hostAliases).
		volumes(volumes).
		// Avoid env vars like <service_name>_PORT="tcp://<ip>:<port>" because they could override regular dogu env vars.
		enableServiceLinks(false).
		initContainers(initContainers...).
		sidecarContainers(r.generateSidecarContainers(doguResource, dogu)...).
		containerEmptyCommandAndArgs().
		containerLivenessProbe().
		containerStartupProbe().
		containerPullPolicy().
		containerVolumeMounts(createVolumeMounts(doguResource, dogu)).
		containerEnvVars(envVars).
		containerResourceRequirements(resourceRequirements).
		serviceAccount().
		securityContext(r.securityContextGenerator.Generate(ctx, dogu, doguResource)).
		build()

	return podTemplate, nil
}

func (r *resourceGenerator) generateInitContainers(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, resourceRequirements corev1.ResourceRequirements) ([]*corev1.Container, error) {
	initContainers := make([]*corev1.Container, 0)
	chownInitImage := r.additionalImages[config.ChownInitImageConfigmapNameKey]

	chownContainer, err := getChownInitContainer(dogu, doguResource, chownInitImage, resourceRequirements)
	if err != nil {
		return nil, err
	}

	initContainers = append(initContainers, chownContainer)

	if hasLocalConfigVolume(dogu) {
		additionalMountsContainerImage := r.additionalImages[config.AdditionalMountsInitContainerImageConfigmapNameKey]
		additionalMountsContainer, err := r.BuildAdditionalMountInitContainer(ctx, dogu, doguResource, additionalMountsContainerImage, resourceRequirements)
		if err != nil {
			return nil, err
		}

		initContainers = append(initContainers, additionalMountsContainer)
	}

	return initContainers, nil
}

func (r *resourceGenerator) generateSidecarContainers(doguResource *k8sv2.Dogu, dogu *core.Dogu) []*corev1.Container {
	sidecars := make([]*corev1.Container, 0)

	if doguResource.Spec.ExportMode {
		exporterImage := r.additionalImages[config.ExporterImageConfigmapNameKey]

		exporterContainer := getExporterContainer(dogu, doguResource, exporterImage)

		sidecars = append(sidecars, exporterContainer)
	}
	return sidecars
}

func hasLocalConfigVolume(dogu *core.Dogu) bool {
	for _, doguVolume := range dogu.Volumes {
		if doguVolume.Name == "localConfig" {
			return true
		}
	}
	return false
}

// findVolumeByName looks for a volume with the given name in the dogu's volumes.
func findVolumeByName(dogu *core.Dogu, volumeName string) (*core.Volume, error) {
	for _, v := range dogu.Volumes {
		if v.Name == volumeName {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("could not find volume name %s in dogu %s", volumeName, dogu.Name)
}

// BuildAdditionalMountInitContainer creates a container for mounting data into a dogu.
func (r *resourceGenerator) BuildAdditionalMountInitContainer(ctx context.Context, dogu *core.Dogu, doguResource *k8sv2.Dogu, image string, requirements corev1.ResourceRequirements) (*corev1.Container, error) {
	mounts, args, err := prepareAdditionalMountsAndArgs(dogu, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare additional mounts configuration: %w", err)
	}

	uid, gid := getUIDAndGIDFromDogu(ctx, dogu)
	runAsNonRoot := false
	readOnlyRootFilesystem := false
	return &corev1.Container{
		Name:            additionalMountsInitContainerName,
		Image:           image,
		Args:            args,
		VolumeMounts:    mounts,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Resources:       requirements,
		// set default values explicitly to make deep equality work
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:  uid,
			RunAsGroup: gid,
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{core.All},
			},
			RunAsNonRoot:           &runAsNonRoot,
			ReadOnlyRootFilesystem: &readOnlyRootFilesystem,
			SELinuxOptions:         &corev1.SELinuxOptions{},
			SeccompProfile:         &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
			AppArmorProfile:        &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeUnconfined},
		},
	}, nil
}

// getUIDAndGIDFromDogu selects the first volume of a dogu and returns the specified uid and gid from it.
// Errors during parsing will be logged and (nil, nil) will be returned.
// We can choose the first volume from the dogu here because in every volume of the dogu.json the ids must be equal.
func getUIDAndGIDFromDogu(ctx context.Context, dogu *core.Dogu) (*int64, *int64) {
	if len(dogu.Volumes) == 0 {
		return nil, nil
	}

	ownerStr := dogu.Volumes[0].Owner
	groupStr := dogu.Volumes[0].Group
	owner, err := strconv.Atoi(ownerStr)
	if err != nil {
		// this only happens if the dogu descriptor is invalid; not much we can do here
		logInvalidVolumePropertyError(ctx, err, "owner", dogu.Name, ownerStr)
		return nil, nil
	}
	group, err := strconv.Atoi(groupStr)
	if err != nil {
		logInvalidVolumePropertyError(ctx, err, "group", dogu.Name, groupStr)
		return nil, nil
	}

	return ptr.To(int64(owner)), ptr.To(int64(group))
}

func logInvalidVolumePropertyError(ctx context.Context, err error, property, doguName, value string) {
	log.FromContext(ctx).Error(err, fmt.Sprintf("dogu-descriptor %q: failed to convert %s %q in volume to int", property, doguName, value))
}

// prepareAdditionalMountsAndArgs generates volume mounts and command arguments for the dogu additional mount init container.
func prepareAdditionalMountsAndArgs(dogu *core.Dogu, doguResource *k8sv2.Dogu) ([]corev1.VolumeMount, []string, error) {
	additionalMounts := doguResource.Spec.AdditionalMounts
	var volumeMounts []corev1.VolumeMount
	args := []string{additionalMountsArg}
	sourceVolumeSet := make(map[string]struct{})

	for _, dataMount := range additionalMounts {
		doguVolume, err := findVolumeByName(dogu, dataMount.Volume)
		if err != nil {
			return nil, nil, err
		}

		// Set up the source volume mount if not already processed
		sourcePath := path.Join(additionalMouuntsDataMountDir, dataMount.Name)
		if _, processed := sourceVolumeSet[dataMount.Name]; !processed {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      dataMount.Name,
				MountPath: sourcePath,
			})
			sourceVolumeSet[dataMount.Name] = struct{}{}
		}

		// Set up init-Container arguments
		targetPath := path.Join(additionalMountsDoguMountDir, doguVolume.Path, dataMount.Subfolder)
		args = append(args, fmt.Sprintf("-source=%s", sourcePath), fmt.Sprintf("-target=%s", targetPath))
	}

	// mount all dogu descriptor volumes as target, so that the deletion of unneeded files is still possible
	volumeMounts = append(volumeMounts, createDoguVolumeMountsWithMountPathPrefix(doguResource, dogu, additionalMountsDoguMountDir)...)
	// add static volumes needed by the init container to write config
	volumeMounts = append(volumeMounts, createStaticDoguConfigVolumeMounts(additionalMountsDoguMountDir)...)

	return volumeMounts, args, nil
}

func getChownInitContainer(dogu *core.Dogu, doguResource *k8sv2.Dogu, chownInitImage string, requirements corev1.ResourceRequirements) (*corev1.Container, error) {
	noInitContainerNeeded := chownInitImage == ""
	if noInitContainerNeeded {
		return nil, nil
	}

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

		isNotRootOwned := uid <= 0 || gid <= 0
		if isNotRootOwned {
			return nil, fmt.Errorf("owner %d or group %d are not greater than 0", uid, gid)
		}

		mkdirCommand := fmt.Sprintf("mkdir -p \"%s\"", volume.Path)
		chownCommand := fmt.Sprintf("chown -R %s:%s \"%s\"", volume.Owner, volume.Group, volume.Path)
		commands = append(commands, mkdirCommand)
		commands = append(commands, chownCommand)
	}

	runAsNonRoot := false
	readOnlyRootFilesystem := false
	return &corev1.Container{
		Name:  chownInitContainerName,
		Image: chownInitImage,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{core.All},
				Add:  []corev1.Capability{core.Chown, core.DacOverride},
			},
			RunAsNonRoot:           &runAsNonRoot,
			ReadOnlyRootFilesystem: &readOnlyRootFilesystem,
			SELinuxOptions:         &corev1.SELinuxOptions{},
			SeccompProfile:         &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
			AppArmorProfile:        &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeUnconfined},
		},
		Command:      []string{"sh", "-c", strings.Join(commands, " && ")},
		VolumeMounts: createDoguVolumeMounts(doguResource, dogu),
		Resources:    requirements,
	}, nil
}

func getExporterContainer(dogu *core.Dogu, doguResource *k8sv2.Dogu, exporterImage string) *corev1.Container {
	exporter := &corev1.Container{
		Name:         CreateExporterContainerName(doguResource.Name),
		Image:        exporterImage,
		VolumeMounts: createExporterSidecarVolumeMounts(doguResource, dogu),
		Env: []corev1.EnvVar{
			{
				Name:  "DOGU_NAME",
				Value: dogu.GetSimpleName(),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{core.All},
				Add:  []corev1.Capability{core.DacOverride, core.SysChroot, core.NetBindService, core.Setgid, core.Setuid},
			},
			SELinuxOptions:  &corev1.SELinuxOptions{},
			SeccompProfile:  &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined},
			AppArmorProfile: &corev1.AppArmorProfile{Type: corev1.AppArmorProfileTypeUnconfined},
		},
	}

	return exporter
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

func updateDeploymentSpec(deployment *appsv1.Deployment, doguResource *k8sv2.Dogu, podTemplate *corev1.PodTemplateSpec) {
	var replicas int32 = ReplicaCountStarted
	if doguResource.Spec.Stopped {
		replicas = ReplicaCountStopped
	}

	deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: doguResource.GetDoguNameLabel()}
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: "Recreate",
	}
	deployment.Spec.Replicas = &replicas
	deployment.Spec.Template = *podTemplate
	deployment.Spec.ProgressDeadlineSeconds = ptr.To(int32(600))
	deployment.Spec.RevisionHistoryLimit = ptr.To(int32(10))
}

// CreateStartupProbe returns a container start-up probe for the given dogu if it contains a state healthcheck.
// Otherwise, it returns nil.
func CreateStartupProbe(dogu *core.Dogu) *corev1.Probe {
	timeoutSeconds := getStartupProbeTimeout()

	for _, healthCheck := range dogu.HealthChecks {
		if healthCheck.Type == "state" {
			state := "ready"
			if healthCheck.State != "" {
				state = healthCheck.State
			}
			return &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					Exec: &corev1.ExecAction{Command: []string{"/bin/sh", "-c", fmt.Sprintf("[ $(doguctl state) = \"%s\" ]", state)}},
				},
				TimeoutSeconds:   timeoutSeconds,
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

func getStartupProbeTimeout() int32 {
	timeoutSeconds := defaultStartupProbeTimeout
	timeoutSecondsStr, found := os.LookupEnv(startupProbeTimoutEnv)
	if found {
		var err error
		timeoutSeconds, err = strconv.Atoi(timeoutSecondsStr)
		if err != nil {
			log.Log.Error(err, fmt.Sprintf("failed to convert dogu startup probe timeout %q to int: defaulting to %q", timeoutSecondsStr, defaultStartupProbeTimeout))
			timeoutSeconds = 1
		}
	}

	return int32(timeoutSeconds)
}

// GetAppLabel returns an app label which all CES resource may receive for general selection.
func GetAppLabel() k8sv2.CesMatchingLabels {
	return map[string]string{appLabelKey: appLabelValueCes}
}

// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
// the service ports to the created service.
func (r *resourceGenerator) CreateDoguService(doguResource *k8sv2.Dogu, dogu *core.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error) {
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

	cesExposedPortAnnotator := annotation.CesExposedPortAnnotator{}
	err = cesExposedPortAnnotator.AnnotateService(service, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to annotate service with exposed ports: %w", err)
	}

	ingressAnnotationCreator := annotation.IngressAnnotator{}
	err = ingressAnnotationCreator.AppendIngressAnnotationsToService(service, doguResource.Spec.AdditionalIngressAnnotations)
	if err != nil {
		return nil, fmt.Errorf("failed to add ingress annotations to service: %w", err)
	}

	err = ctrl.SetControllerReference(doguResource, service, r.scheme)
	if err != nil {
		return nil, wrapControllerReferenceError(err)
	}

	return service, nil
}

func wrapControllerReferenceError(err error) error {
	return fmt.Errorf("failed to set controller reference: %w", err)
}

// CreateExporterContainerName creates the name for the exporter-container used as a sidecar-container
func CreateExporterContainerName(doguName string) string {
	return fmt.Sprintf("%s-exporter", doguName)
}
