package exec

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/retry"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// execPod provides features to handle files from a dogu image.
type execPod struct {
	client     client.Client
	executor   commandExecutor
	volumeMode PodVolumeMode

	doguResource *k8sv1.Dogu
	dogu         *core.Dogu
	podName      string
	deleteSpec   *corev1.Pod
}

// NewExecPod creates a new ExecPod that enables command execution towards a pod.
func NewExecPod(
	client client.Client,
	executor commandExecutor,
	factoryMode PodVolumeMode,
	doguResource *k8sv1.Dogu,
	dogu *core.Dogu,
	podName string,
) (*execPod, error) {
	return &execPod{
		client:       client,
		executor:     executor,
		volumeMode:   factoryMode,
		doguResource: doguResource,
		dogu:         dogu,
		podName:      podName,
	}, nil
}

// Create adds a new exec pod to the cluster. It waits synchronously until the K8s pod resource exists.
func (ep *execPod) Create(ctx context.Context) error {
	logger := log.FromContext(ctx)

	execPodSpec, err := ep.createPod(ctx, ep.doguResource.Namespace, ep.podName)
	if err != nil {
		return err
	}
	ep.deleteSpec = execPodSpec

	logger.Info("Creating new exec pod " + ep.podName)
	err = ep.client.Create(ctx, execPodSpec)
	if err != nil {
		return err
	}

	err = ep.waitForPodToSpawn(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (ep *execPod) createPod(ctx context.Context, k8sNamespace string, containerName string) (*corev1.Pod, error) {
	image := ep.dogu.Image + ":" + ep.dogu.Version
	// command is of no importance because the pod will be killed after success
	doNothingCommand := []string{"/bin/sleep", "60"}
	// set app name for completeness's sake so all generated resource can be selected (and possibly cleaned up) with our ces label.
	appLabels := resource.GetAppLabel()

	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

	volumeMounts, volumes := ep.createVolumes(ctx)

	podSpec := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        containerName,
			Namespace:   k8sNamespace,
			Labels:      appLabels,
			Annotations: make(map[string]string),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            containerName,
					Image:           image,
					Command:         doNothingCommand,
					ImagePullPolicy: pullPolicy,
					VolumeMounts:    volumeMounts,
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "k8s-dogu-operator-docker-registry"},
			},
			Volumes: volumes,
		},
	}

	err := ctrl.SetControllerReference(ep.doguResource, podSpec, ep.client.Scheme())
	if err != nil {
		return nil, fmt.Errorf("failed to set controller reference to exec pod %s: %w", containerName, err)
	}
	return podSpec, nil
}

func (ep *execPod) createVolumes(ctx context.Context) ([]corev1.VolumeMount, []corev1.Volume) {
	logger := log.FromContext(ctx)

	switch ep.volumeMode {
	case PodVolumeModeInstall:
		return nil, nil
	case PodVolumeModeUpgrade:
		volumeMounts := []corev1.VolumeMount{{
			Name:      ep.doguResource.GetReservedVolumeName(),
			ReadOnly:  false,
			MountPath: resource.DoguReservedPath,
		}}

		volumes := []corev1.Volume{{
			Name: ep.doguResource.GetReservedVolumeName(),
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: ep.doguResource.GetReservedPVCName(),
				},
			},
		}}
		return volumeMounts, volumes
	}
	logger.Info("ExecPod is about to be created without volumes because of unexpected factory mode %d", ep.volumeMode)
	return nil, nil
}

func (ep *execPod) waitForPodToSpawn(ctx context.Context) error {
	logger := log.FromContext(ctx)

	execPodKey := ep.ObjectKey()
	containerPodName := execPodKey.Name

	err := retry.OnErrorRetry(maxTries, retry.TestableRetryFunc, func() error {
		lePod, err := ep.getPod(ctx)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Error while finding exec pod %s. Trying again...", containerPodName))
			return &retry.TestableRetrierError{Err: err}
		}

		leStatus := lePod.Status.Phase
		switch leStatus {
		case corev1.PodRunning:
			logger.Info("Found a ready exec pod " + containerPodName)
			return nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("quitting dogu installation because exec pod %s failed with status %s or did not come up in time", containerPodName, leStatus)
		default:
			logger.Info(fmt.Sprintf("Found exec pod %s but with status phase %+v. Trying again...", containerPodName, leStatus))
			return &retry.TestableRetrierError{Err: fmt.Errorf("found exec pod %s but with status phase %+v", containerPodName, leStatus)}
		}
	})
	if err != nil {
		return fmt.Errorf("failed to wait for exec pod %s to spawn: %w", containerPodName, err)
	}

	return nil
}

func (ep *execPod) getPod(ctx context.Context) (*corev1.Pod, error) {
	lePod := &corev1.Pod{}
	err := ep.client.Get(ctx, *ep.ObjectKey(), lePod)

	return lePod, err
}

// Delete deletes the exec pod from the cluster.
func (ep *execPod) Delete(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("Cleaning up exec pod ", ep.podName)
	err := ep.client.Delete(ctx, ep.deleteSpec)
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete execPod %s: %w", ep.podName, err)
		}

		logger.Error(err, fmt.Sprintf("Could not find execPod %s for deletion", ep.podName))
	}

	return nil
}

// PodName returns the name of the created exec pod resource.
func (ep *execPod) PodName() string {
	return ep.podName
}

// ObjectKey returns an execPod's K8s object key.
func (ep *execPod) ObjectKey() *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: ep.doguResource.Namespace,
		Name:      ep.podName,
	}
}

// Exec executes the given ShellCommand and returns any output to stdOut and stdErr.
func (ep *execPod) Exec(ctx context.Context, cmd *ShellCommand) (string, error) {
	pod, err := ep.getPod(ctx)
	if err != nil {
		return "", fmt.Errorf("could not get pod: %w", err)
	}

	out, err := ep.executor.ExecCommandForPod(ctx, pod, cmd, ContainersStarted)

	return out.String(), err
}

type defaultSufficeGenerator struct{}

// String returns a pod suffix of fixed length.
func (sg *defaultSufficeGenerator) String(suffixLength int) string {
	return rand.String(suffixLength)
}

// PodVolumeMode indicates whether to mount a dogu's PVC (which only makes sense when the dogu was already
// installed).
type PodVolumeMode int

const (
	// PodVolumeModeInstall indicates to not mount a dogu's PVC.
	PodVolumeModeInstall PodVolumeMode = iota
	// PodVolumeModeUpgrade indicates to mount a dogu's PVC.
	PodVolumeModeUpgrade
)

type defaultExecPodFactory struct {
	client          client.Client
	config          *rest.Config
	commandExecutor commandExecutor
	suffixGen       suffixGenerator
}

// NewExecPodFactory creates a new ExecPodFactory.
func NewExecPodFactory(client client.Client, config *rest.Config, executor commandExecutor) *defaultExecPodFactory {
	return &defaultExecPodFactory{
		client:          client,
		config:          config,
		commandExecutor: executor,
		suffixGen:       &defaultSufficeGenerator{},
	}
}

// NewExecPod creates a new ExecPod during the operation run-time.
func (epf *defaultExecPodFactory) NewExecPod(execPodFactoryMode PodVolumeMode, doguResource *k8sv1.Dogu, dogu *core.Dogu) (ExecPod, error) {
	podName := generatePodName(dogu, epf.suffixGen)
	return NewExecPod(epf.client, epf.commandExecutor, execPodFactoryMode, doguResource, dogu, podName)
}

func generatePodName(dogu *core.Dogu, generator suffixGenerator) string {
	return fmt.Sprintf("%s-%s-%s", dogu.GetSimpleName(), "execpod", generator.String(6))
}
