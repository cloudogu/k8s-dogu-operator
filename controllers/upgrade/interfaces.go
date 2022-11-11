package upgrade

import (
	"bytes"
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
)

type imageRegistry interface {
	// PullImageConfig pulls a given container image by name.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into a map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, execpod exec.ExecPod) (map[string]string, error)
}

type serviceAccountCreator interface {
	// CreateAll creates K8s services accounts for a dogu
	CreateAll(ctx context.Context, dogu *core.Dogu) error
}

type doguRegistrator interface {
	// RegisterDoguVersion registers a certain dogu in a CES instance.
	RegisterDoguVersion(dogu *core.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}

type resourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except customDeployment which may be nil.
	UpsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *imagev1.ConfigFile) (*corev1.Service, error)
	// UpsertDoguExposedServices creates exposed services based on the given dogu. If an error occurs during creating
	// several exposed services, this method tries to apply as many exposed services as possible and returns then
	// an error collection.
	UpsertDoguExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]*corev1.Service, error)
	// UpsertDoguPVCs generates a persitent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (*corev1.PersistentVolumeClaim, error)
}

type execPodFactory interface {
	// NewExecPod creates a new ExecPod.
	NewExecPod(execPodFactoryMode exec.PodVolumeMode, doguResource *k8sv1.Dogu, dogu *core.Dogu) (exec.ExecPod, error)
}

// commandExecutor is used to execute commands in pods and dogus
type commandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, resource *k8sv1.Dogu, command *exec.ShellCommand, expectedStatus exec.PodStatus) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command *exec.ShellCommand, expectedStatus exec.PodStatus) (*bytes.Buffer, error)
}
