package resource

import (
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type podSpecBuilder struct {
	theDoguResource                  *k8sv2.Dogu
	theDogu                          *core.Dogu
	metaAllLabels                    k8sv2.CesMatchingLabels
	metaAllAnnotations               map[string]string
	specHostAliases                  []corev1.HostAlias
	specVolumes                      []corev1.Volume
	specEnableServiceLinks           bool
	specServiceAccountName           string
	specAutomountServiceAccountToken bool
	specInitContainers               []corev1.Container
	specSidecarContainers            []corev1.Container
	specContainerCommand             []string
	specContainerArgs                []string
	specContainerLivenessProbe       *corev1.Probe
	specContainerStartupProbe        *corev1.Probe
	specContainerImagePullPolicy     corev1.PullPolicy
	specContainerVolumeMounts        []corev1.VolumeMount
	specContainerEnvVars             []corev1.EnvVar
	specContainerResourcesReq        corev1.ResourceRequirements
	specPodSecurityContext           *corev1.PodSecurityContext
	specContainerSecurityContext     *corev1.SecurityContext
}

func newPodSpecBuilder(doguResource *k8sv2.Dogu, dogu *core.Dogu) *podSpecBuilder {
	p := &podSpecBuilder{}
	p.theDoguResource = doguResource
	p.theDogu = dogu
	return p
}

func (p *podSpecBuilder) labels(labels k8sv2.CesMatchingLabels) *podSpecBuilder {
	p.metaAllLabels = labels
	return p
}

func (p *podSpecBuilder) annotations(annos map[string]string) *podSpecBuilder {
	p.metaAllAnnotations = annos
	return p
}

func (p *podSpecBuilder) hostAliases(hostAliases []corev1.HostAlias) *podSpecBuilder {
	p.specHostAliases = hostAliases
	return p
}

func (p *podSpecBuilder) volumes(volumes []corev1.Volume) *podSpecBuilder {
	p.specVolumes = volumes
	return p
}

func (p *podSpecBuilder) enableServiceLinks(enable bool) *podSpecBuilder {
	p.specEnableServiceLinks = enable
	return p
}

func (p *podSpecBuilder) initContainers(initContainers ...*corev1.Container) *podSpecBuilder {
	for _, initContainer := range initContainers {
		if initContainer == nil {
			continue
		}

		foundContainer := *initContainer
		p.specInitContainers = append(p.specInitContainers, foundContainer)
	}
	return p
}

func (p *podSpecBuilder) sidecarContainers(sidecarContainers ...*corev1.Container) *podSpecBuilder {
	for _, sidecarContainer := range sidecarContainers {
		if sidecarContainer == nil {
			continue
		}

		foundContainer := *sidecarContainer
		p.specSidecarContainers = append(p.specSidecarContainers, foundContainer)
	}
	return p
}

// containerEmptyCommandAndArgs adds empty container commands and arguments to the template spec. These are by default
// empty so the container uses its own RUN or ENTRYPOINT instruction. These can be overridden with custom commands,
// f. i. to start the support mode during a container crash loop.
func (p *podSpecBuilder) containerEmptyCommandAndArgs() *podSpecBuilder {
	var empty []string
	p.specContainerCommand = empty
	p.specContainerArgs = empty

	return p
}

func (p *podSpecBuilder) containerLivenessProbe() *podSpecBuilder {
	for _, healthCheck := range p.theDogu.HealthChecks {
		if healthCheck.Type == "tcp" {
			p.specContainerLivenessProbe = &corev1.Probe{
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

			return p
		}
	}

	return p
}

func (p *podSpecBuilder) containerStartupProbe() *podSpecBuilder {
	p.specContainerStartupProbe = CreateStartupProbe(p.theDogu)

	return p
}

func (p *podSpecBuilder) containerPullPolicy() *podSpecBuilder {
	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	p.specContainerImagePullPolicy = pullPolicy

	return p
}

func (p *podSpecBuilder) containerVolumeMounts(volumeMounts []corev1.VolumeMount) *podSpecBuilder {
	p.specContainerVolumeMounts = volumeMounts
	return p
}

func (p *podSpecBuilder) containerEnvVars(envVars []corev1.EnvVar) *podSpecBuilder {
	p.specContainerEnvVars = envVars
	return p
}

func (p *podSpecBuilder) containerResourceRequirements(reqs corev1.ResourceRequirements) *podSpecBuilder {
	p.specContainerResourcesReq = reqs
	return p
}

func (p *podSpecBuilder) serviceAccount() *podSpecBuilder {
	for _, account := range p.theDogu.ServiceAccounts {
		if account.Kind == kubernetesServiceAccountKind && account.Type == doguOperatorClient {
			// this is used for service accounts deployed from the k8s-folder of a dogu, e.g. in nginx-ingress
			p.specServiceAccountName = p.theDogu.GetSimpleName()
			p.specAutomountServiceAccountToken = true
			return p
		}
	}

	return p
}

func (p *podSpecBuilder) securityContext(podSecurityContext *corev1.PodSecurityContext, containerSecurityContext *corev1.SecurityContext) *podSpecBuilder {
	p.specPodSecurityContext = podSecurityContext
	p.specContainerSecurityContext = containerSecurityContext

	return p
}

func (p *podSpecBuilder) build() *corev1.PodTemplateSpec {
	result := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      p.metaAllLabels,
			Annotations: p.metaAllAnnotations,
		},
		Spec: corev1.PodSpec{
			ImagePullSecrets:             []corev1.LocalObjectReference{{Name: "ces-container-registries"}},
			Hostname:                     p.theDoguResource.Name,
			HostAliases:                  p.specHostAliases,
			Volumes:                      p.specVolumes,
			EnableServiceLinks:           &p.specEnableServiceLinks,
			ServiceAccountName:           p.specServiceAccountName,
			AutomountServiceAccountToken: &p.specAutomountServiceAccountToken,
			InitContainers:               p.specInitContainers,
			SecurityContext:              p.specPodSecurityContext,
			Containers:                   p.buildContainers(),
		},
	}

	return result
}

func (p *podSpecBuilder) buildContainers() []corev1.Container {
	containers := []corev1.Container{p.buildDoguContainer()}

	return append(containers, p.specSidecarContainers...)
}

func (p *podSpecBuilder) buildDoguContainer() corev1.Container {
	container := corev1.Container{
		Name:            p.theDoguResource.Name,
		Image:           p.theDogu.Image + ":" + p.theDogu.Version,
		Command:         p.specContainerCommand,
		Args:            p.specContainerArgs,
		LivenessProbe:   p.specContainerLivenessProbe,
		StartupProbe:    p.specContainerStartupProbe,
		ImagePullPolicy: p.specContainerImagePullPolicy,
		VolumeMounts:    p.specContainerVolumeMounts,
		Env:             p.specContainerEnvVars,
		Resources:       p.specContainerResourcesReq,
		SecurityContext: p.specContainerSecurityContext,
	}

	return container
}
