package resource

import (
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/internal"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/util/json"
	"strconv"
	"strings"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
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

// resourceGenerator generate k8s resources for a given dogu. All resources will be referenced with the dogu resource
// as controller
type resourceGenerator struct {
	scheme           *runtime.Scheme
	doguLimitPatcher internal.LimitPatcher
}

// NewResourceGenerator creates a new generator for k8s resources
func NewResourceGenerator(scheme *runtime.Scheme, limitPatcher internal.LimitPatcher) *resourceGenerator {
	return &resourceGenerator{scheme: scheme, doguLimitPatcher: limitPatcher}
}

// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
// The deploymentPatch is applied at the end of resource generation.
func (r *resourceGenerator) CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu, deploymentPatch func(*appsv1.Deployment)) (*appsv1.Deployment, error) {
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

	if deploymentPatch != nil {
		deploymentPatch(deployment)
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
	volumes, err := createVolumesForDogu(doguResource, dogu)
	if err != nil {
		return nil, err
	}

	volumeMounts := createVolumeMountsForDogu(doguResource, dogu)
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

	podTemplate := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: allLabels,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "k8s-dogu-operator-docker-registry"}},
			Hostname:         doguResource.Name,
			Volumes:          volumes,
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

func getKubernetesServiceAccount(dogu *core.Dogu) (string, bool) {
	for _, account := range dogu.ServiceAccounts {
		if account.Kind == string(k8sv1.KubernetesServiceAccountKind) && account.Type == k8sv1.DoguOperatorClient {
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

func createVolumesForDogu(doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]corev1.Volume, error) {
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

	// always reserve a volume for upgrade script actions, even if the dogu has no state because upgrade scripts
	// do not always rely on a dogu state (f. e. checks on upgradability)
	doguReservedVolume := corev1.Volume{
		Name: doguResource.GetReservedVolumeName(),
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: doguResource.GetReservedPVCName(),
			},
		},
	}

	volumes := []corev1.Volume{
		nodeMasterVolume,
		privateVolume,
		doguReservedVolume,
	}

	volumesFromDogu, err := createVolumesFromDoguVolumes(dogu.Volumes, doguResource)
	if err != nil {
		return nil, err
	}

	volumes = append(volumes, volumesFromDogu...)

	return volumes, nil
}

func createVolumesFromDoguVolumes(doguVolumes []core.Volume, doguResource *k8sv1.Dogu) ([]corev1.Volume, error) {
	var multiError error
	var volumes []corev1.Volume
	for _, doguVolume := range doguVolumes {
		_, clientExists := doguVolume.GetClient(k8sv1.DoguOperatorClient)
		if clientExists {
			volume, err := createClientVolumeFromDoguVolume(doguVolume)
			if err != nil {
				multiError = multierror.Append(multiError, err)
				continue
			}

			volumes = append(volumes, *volume)
		}
	}

	dataVolumeCount := len(doguVolumes) - len(volumes)
	if dataVolumeCount > 0 {
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

	return volumes, multiError
}

func createClientVolumeFromDoguVolume(doguVolume core.Volume) (*corev1.Volume, error) {
	client, clientExists := doguVolume.GetClient(k8sv1.DoguOperatorClient)
	if !clientExists {
		return nil, fmt.Errorf("dogu volume %s has no client", doguVolume.Name)
	}

	clientParams := new(k8sv1.VolumeParams)
	err := convertGenericJsonObject(client.Params, clientParams)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s client params of volume %s: %w", k8sv1.DoguOperatorClient, doguVolume.Name, err)
	}

	switch clientParams.Type {
	case k8sv1.ConfigMapParamType:
		configMapParamContent := new(k8sv1.VolumeConfigMapContent)
		err = convertGenericJsonObject(clientParams.Content, configMapParamContent)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s client type content of volume %s: %w", k8sv1.ConfigMapParamType, doguVolume.Name, err)
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
			Name:      doguResource.GetReservedVolumeName(),
			ReadOnly:  false,
			MountPath: DoguReservedPath,
		},
	}

	for _, doguVolume := range dogu.Volumes {
		newVolume := createVolumeMountFromDoguVolume(doguVolume, doguResource)
		doguVolumeMounts = append(doguVolumeMounts, newVolume)
	}

	return doguVolumeMounts
}

func createVolumeMountFromDoguVolume(doguVolume core.Volume, doguResource *k8sv1.Dogu) corev1.VolumeMount {
	_, clientExists := doguVolume.GetClient(k8sv1.DoguOperatorClient)
	if clientExists {
		return corev1.VolumeMount{
			Name:      doguVolume.Name,
			ReadOnly:  false,
			MountPath: doguVolume.Path,
		}
	}

	return corev1.VolumeMount{
		Name:      doguResource.GetDataVolumeName(),
		ReadOnly:  false,
		MountPath: doguVolume.Path,
		SubPath:   doguVolume.Name,
	}
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

// CreateDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu.
func (r *resourceGenerator) CreateDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error) {
	return r.createPVC(doguResource.Name, doguResource, resource.MustParse("5Gi"))
}

// CreateReservedPVC creates a persistent volume claim with a 10Mi storage for the given dogu.
// Used for example for upgrade operations.
func (r *resourceGenerator) CreateReservedPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error) {
	return r.createPVC(doguResource.GetReservedPVCName(), doguResource, resource.MustParse("10Mi"))
}

func (r *resourceGenerator) createPVC(pvcName string, doguResource *k8sv1.Dogu, size resource.Quantity) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: doguResource.Namespace,
			Labels:    GetAppLabel().Add(doguResource.GetDoguNameLabel()),
		},
	}

	pvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.ResourceRequirements{
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

// CreateDoguSecret generates a secret with a given data map for the dogu
func (r *resourceGenerator) CreateDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error) {
	appDoguLabels := GetAppLabel().Add(doguResource.GetDoguNameLabel())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.GetPrivateVolumeName(),
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
