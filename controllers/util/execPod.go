package util

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ExecPod provides methods for instantiating and removing an intermediate pod based on a Dogu container image.
type ExecPod interface {
	// Create adds a new exec pod to the cluster.
	Create(ctx context.Context) error
	// Delete deletes the exec pod from the cluster.
	Delete(ctx context.Context) error
	// PodName returns the name of the pod.
	PodName() string
	// ObjectKey returns the ExecPod's K8s object key.
	ObjectKey() *client.ObjectKey
	// Exec runs the provided command in this execPod
	Exec(ctx context.Context, cmd *resource.ShellCommand) (out string, err error)
}

// maxTries controls the maximum number of waiting intervals between requesting an exec pod and its actual
// instantiation. The waiting time linearly increases each iteration.
var maxTries = 20

type suffixGenerator interface {
	// String returns a random suffix string with the given length
	String(length int) string
}

// commandExecutor is used to execute command in a dogu
type commandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, targetDogu string, namespace string, command *resource.ShellCommand) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, podName string, namespace string, command *resource.ShellCommand) (*bytes.Buffer, error)
}

// execPod provides features to handle files from a dogu image.
type execPod struct {
	client      client.Client
	executor    commandExecutor
	factoryMode ExecPodVolumeMode

	doguResource *k8sv1.Dogu
	dogu         *core.Dogu
	podName      string
	deleteSpec   *corev1.Pod
}

// NewExecPod creates a new ExecPod that enables command execution towards a pod.
func NewExecPod(client client.Client, restConfig *rest.Config, factoryMode ExecPodVolumeMode, doguResource *k8sv1.Dogu, dogu *core.Dogu, podName string) (*execPod, error) {
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())

	return &execPod{
		client:       client,
		executor:     executor,
		factoryMode:  factoryMode,
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
	labels := map[string]string{"app": "ces", "dogu": containerName}

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
			Labels:      labels,
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

	switch ep.factoryMode {
	case ExecPodVolumeModeInstall:
		return nil, nil
	case ExecPodVolumeModeUpgrade:
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
	logger.Info("ExecPod is about to be created without volumes because of unexpected factory mode %d", ep.factoryMode)
	return nil, nil
}

func (ep *execPod) waitForPodToSpawn(ctx context.Context) error {
	logger := log.FromContext(ctx)

	execPodKey := ep.ObjectKey()

	lePod := corev1.Pod{}
	containerPodName := execPodKey.Name

	for i := 1; i <= maxTries; i++ {
		if i >= maxTries {
			return fmt.Errorf("quitting dogu installation because exec pod %s could not be found", containerPodName)
		}

		err := ep.client.Get(ctx, *execPodKey, &lePod)
		if err != nil {
			logger.Error(err, "Error while finding exec pod "+containerPodName+". Will try again.")
			sleep(logger, i)
			continue
		}

		leStatus := lePod.Status.Phase
		switch leStatus {
		case corev1.PodRunning:
			logger.Info("Found a ready exec pod " + containerPodName)
			return nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return fmt.Errorf("quitting dogu installation because exec pod %s failed with status %s or did not come up in time", containerPodName, leStatus)
		default:
			logger.Info(fmt.Sprintf("Found exec pod %s but with status phase %+v", containerPodName, leStatus))
			sleep(logger, i)
			continue
		}
	}

	return fmt.Errorf("unexpected loop end while finding exec pod %s", containerPodName)
}

// Delete deletes the exec pod from the cluster.
func (ep *execPod) Delete(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("Cleaning up exec pod ", ep.podName)
	err := ep.client.Delete(ctx, ep.deleteSpec)
	if err != nil {
		err2 := fmt.Errorf("failed to delete custom dogu descriptor: %w", err)
		if !errors.IsNotFound(err) {
			return err2
		}

		logger.Error(err2, "Error deleting execPod ")
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
func (ep *execPod) Exec(ctx context.Context, cmd *resource.ShellCommand) (string, error) {
	out, err := ep.executor.ExecCommandForPod(ctx, ep.podName, ep.doguResource.Namespace, cmd)
	return out.String(), err
}

func sleep(logger logr.Logger, sleepIntervalInSec int) {
	logger.Info(fmt.Sprintf("Exec pod not found. Trying again in %d second(s)", sleepIntervalInSec))
	time.Sleep(time.Duration(sleepIntervalInSec) * time.Second) // linear rising backoff
}

type defaultSufficeGenerator struct{}

// String returns a pod suffix of fixed length.
func (sg *defaultSufficeGenerator) String(suffixLength int) string {
	return rand.String(suffixLength)
}

// ExecPodVolumeMode indicates whether to mount a dogu's PVC (which only makes sense when the dogu was already
// installed).
type ExecPodVolumeMode int

const (
	// ExecPodVolumeModeInstall indicates to not mount a dogu's PVC.
	ExecPodVolumeModeInstall ExecPodVolumeMode = iota
	// ExecPodVolumeModeUpgrade indicates to mount a dogu's PVC.
	ExecPodVolumeModeUpgrade
)

type defaultExecPodFactory struct {
	client    client.Client
	config    *rest.Config
	suffixGen suffixGenerator
}

// NewExecPodFactory creates a new ExecPodFactory.
func NewExecPodFactory(client client.Client, config *rest.Config) *defaultExecPodFactory {
	return &defaultExecPodFactory{
		client:    client,
		config:    config,
		suffixGen: &defaultSufficeGenerator{},
	}
}

// NewExecPod creates a new ExecPod during the operation run-time.
func (epf *defaultExecPodFactory) NewExecPod(execPodFactoryMode ExecPodVolumeMode, doguResource *k8sv1.Dogu, dogu *core.Dogu) (ExecPod, error) {
	podName := generatePodName(dogu, epf.suffixGen)
	return NewExecPod(epf.client, epf.config, execPodFactoryMode, doguResource, dogu, podName)
}

func generatePodName(dogu *core.Dogu, generator suffixGenerator) string {
	return fmt.Sprintf("%s-%s-%s", dogu.GetSimpleName(), "execpod", generator.String(6))
}
