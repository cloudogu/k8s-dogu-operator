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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/scheme"
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
	Exec(cmd *resource.ShellCommand) (stdOut string, errOut string, err error)
}

// maxTries controls the maximum number of waiting intervals between requesting an exec pod and its actual
// instantiation. The waiting time linearly increases each iteration.
var maxTries = 20

type suffixGenerator interface {
	// String returns a random suffix string with the given length
	String(length int) string
}

// execPod provides features to handle files from a dogu image.
type execPod struct {
	client   client.Client
	executor commandExecutor

	doguResource *k8sv1.Dogu
	dogu         *core.Dogu
	podName      string
	deleteSpec   *corev1.Pod
}

// NewExecPod creates a new ExecPod that enables command execution towards a pod.
func NewExecPod(client client.Client, restConfig rest.Config, doguResource *k8sv1.Dogu, dogu *core.Dogu, podName string) (*execPod, error) {
	// restConfig is not a pointer because we modify it here
	restConfig.APIPath = "/api"
	restConfig.GroupVersion = &schema.GroupVersion{Version: "v1"}
	restConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	executor, err := NewCommandExecutor(podName, podName, doguResource.Namespace, &restConfig)
	if err != nil {
		return nil, err
	}
	return &execPod{
		client:       client,
		executor:     executor,
		doguResource: doguResource,
		dogu:         dogu,
		podName:      podName,
	}, nil
}

// Create adds a new exec pod to the cluster. It waits synchronously until the K8s pod resource exists.
func (ep *execPod) Create(ctx context.Context) error {
	logger := log.FromContext(ctx)

	execPodSpec, err := ep.createPod(ep.doguResource.Namespace, ep.podName)
	if err != nil {
		return err
	}

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

func (ep *execPod) createPod(k8sNamespace string, containerName string) (*corev1.Pod, error) {
	image := ep.dogu.Image + ":" + ep.dogu.Version
	// command is of no importance because the pod will be killed after success
	doNothingCommand := []string{"/bin/sleep", "60"}
	labels := map[string]string{"app": "ces", "dogu": containerName}

	pullPolicy := corev1.PullIfNotPresent
	if config.Stage == config.StageDevelopment {
		pullPolicy = corev1.PullAlways
	}

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
					VolumeMounts: []corev1.VolumeMount{{
						Name:      resource.DoguReservedVolume,
						ReadOnly:  false,
						MountPath: resource.DoguReservedPath,
					}},
				},
			},
			ImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "k8s-dogu-operator-docker-registry"},
			},
			Volumes: []corev1.Volume{{
				Name: resource.DoguReservedVolume,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ep.doguResource.Name,
						ReadOnly:  false,
					},
				},
			}},
		},
	}

	err := ctrl.SetControllerReference(ep.doguResource, podSpec, ep.client.Scheme())
	if err != nil {
		return nil, fmt.Errorf("failed to set controller reference to exec pod %s: %w", containerName, err)
	}
	return podSpec, nil
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
func (ep *execPod) Exec(cmd *resource.ShellCommand) (stdOut string, errOut string, err error) {
	outBytes, errOutBytes, err := ep.executor.ExecCmd(cmd)
	return outBytes.String(), errOutBytes.String(), err
}

// commandExecutor provides the functionality to execute a shell command in a pod.
type commandExecutor interface {
	// ExecCmd executes the given ShellCommand.
	ExecCmd(cmd *resource.ShellCommand) (out, errOut *bytes.Buffer, err error)
}

type defaultCommandExecutor struct {
	runner runner
}

// NewCommandExecutor creates a new command executor.
func NewCommandExecutor(podName string, containerName string, namespace string, restConfig *rest.Config) (*defaultCommandExecutor, error) {
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	options := &runWrapper{
		ExecOptions: &exec.ExecOptions{
			StreamOptions: exec.StreamOptions{
				Namespace:       namespace,
				PodName:         podName,
				ContainerName:   containerName,
				Stdin:           true,
				TTY:             false,
				Quiet:           false,
				InterruptParent: nil,
				IOStreams:       createStreams(),
			},
			Executor:      &exec.DefaultRemoteExecutor{},
			PodClient:     clientSet.CoreV1(),
			GetPodTimeout: 0,
			Config:        restConfig,
		},
	}

	return &defaultCommandExecutor{
		runner: options,
	}, nil
}

type runner interface {
	// Run executes a command that was provided by SetCommand()
	Run() (genericclioptions.IOStreams, error)
	// SetCommand fills a ShellCommand during run-time in order to execute it afterwards.
	SetCommand(command *resource.ShellCommand)
}

type runWrapper struct {
	*exec.ExecOptions
}

// Run executes a command that was provided by SetCommand()
func (r *runWrapper) Run() (genericclioptions.IOStreams, error) {
	err := r.ExecOptions.Run()
	return r.IOStreams, err
}

// SetCommand fills a ShellCommand during run-time in order to execute it afterwards.
func (r *runWrapper) SetCommand(command *resource.ShellCommand) {
	r.Command = append([]string{command.Command}, command.Args...)
}

// ExecCmd executes arbitrary commands in a pod container.
func (ce *defaultCommandExecutor) ExecCmd(cmd *resource.ShellCommand) (out, errOut *bytes.Buffer, err error) {
	ce.runner.SetCommand(cmd)

	streams, err := ce.runner.Run()

	return streams.Out.(*bytes.Buffer),
		streams.ErrOut.(*bytes.Buffer),
		err
}

func createStreams() genericclioptions.IOStreams {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	return genericclioptions.IOStreams{
		In:     in,
		Out:    out,
		ErrOut: errOut,
	}
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
func (epf *defaultExecPodFactory) NewExecPod(doguResource *k8sv1.Dogu, dogu *core.Dogu) (ExecPod, error) {
	podName := generatePodName(dogu, epf.suffixGen)
	return NewExecPod(epf.client, *epf.config, doguResource, dogu, podName)
}

func generatePodName(dogu *core.Dogu, generator suffixGenerator) string {
	return fmt.Sprintf("%s-%s-%s", dogu.GetSimpleName(), "execpod", generator.String(6))
}
