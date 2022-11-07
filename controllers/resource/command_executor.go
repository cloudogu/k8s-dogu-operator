package resource

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ShellCommand represents all necessary arguments to execute a command inside a container.
type ShellCommand struct {
	// Command states the actual executable that is supposed to be executed in the container.
	Command string
	// Args contains any parameters, switches etc. that the command needs to run properly.
	Args []string
}

// NewShellCommand creates a new ShellCommand. While the command is mandatory, there can be zero to n command arguments.
func NewShellCommand(command string, args ...string) *ShellCommand {
	return &ShellCommand{Command: command, Args: args}
}

func (sc *ShellCommand) String() string {
	result := []string{sc.Command}
	return strings.Join(append(result, sc.Args...), " ")
}

// stateError is returned when a specific resource (pod/dogu) does not meet the requirements for the exec.
type stateError struct {
	sourceError error
	resource    metav1.Object
}

// Report returns the error in string representation
func (e *stateError) Error() string {
	return fmt.Sprintf("resource does not meet requirements for exec: %v, source error: %s", e.resource.GetName(), e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *stateError) Requeue() bool {
	return true
}

// maxTries controls the maximum number of waiting intervals between tries when getting an error that is recoverable
// during command execution.
var maxTries = 20

// PodStatus describes a state in the lifecycle of a pod.
type PodStatus string

const (
	// ContainersStarted means that all containers of a pod were started.
	ContainersStarted PodStatus = "started"
	// PodReady means that the readiness probe of the pod has succeeded.
	PodReady PodStatus = "ready"
)

// commandExecutor is the unit to execute commands in a dogu
type commandExecutor struct {
	Client                 client.Client
	ClientSet              kubernetes.Interface
	CoreV1RestClient       rest.Interface
	CommandExecutorCreator func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)
}

// NewCommandExecutor creates a new instance of NewCommandExecutor
func NewCommandExecutor(cli client.Client, clientSet kubernetes.Interface, coreV1RestClient rest.Interface) *commandExecutor {
	return &commandExecutor{
		Client:    cli,
		ClientSet: clientSet,
		// the rest clientSet COULD be generated from the clientSet but makes harder to test, so we source it additionally
		CoreV1RestClient:       coreV1RestClient,
		CommandExecutorCreator: remotecommand.NewSPDYExecutor,
	}
}

// ExecCommandForDogu execs a command in the first found pod of a dogu. This method executes a command on a dogu pod
// that can be selected by a K8s label.
func (ce *commandExecutor) ExecCommandForDogu(ctx context.Context, resource *v1.Dogu, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error) {
	logger := log.FromContext(ctx)
	pod := &corev1.Pod{}
	err := util.OnErrorRetryAlways(maxTries, func() error {
		var err error
		pod, err = resource.GetPod(ctx, ce.Client)
		if err != nil {
			logger.Info(fmt.Sprintf("Failed to get pod. Trying again: %s", err.Error()))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod for dogu %s: %w", resource.Name, err)
	}

	return ce.ExecCommandForPod(ctx, pod, command, expectedStatus)
}

// ExecCommandForPod execs a command in a given pod. This method executes a command on an arbitrary pod that can be
// identified by its pod name.
func (ce *commandExecutor) ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error) {
	err := ce.waitForPodToHaveExpectedStatus(pod, expectedStatus)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while waiting for pod %s to have status %s: %w", pod.Name, expectedStatus, err)
	}

	req := ce.getCreateExecRequest(pod, command)
	exec, err := ce.CommandExecutorCreator(ctrl.GetConfigOrDie(), "POST", req.URL())
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("failed to create new spdy executor: %w", err),
			resource:    pod,
		}
	}

	return ce.streamCommandToPod(ctx, exec, command, pod)
}

func (ce *commandExecutor) streamCommandToPod(
	ctx context.Context,
	exec remotecommand.Executor,
	command *ShellCommand,
	pod *corev1.Pod,
) (*bytes.Buffer, error) {
	logger := log.FromContext(ctx)

	var err error
	buffer := bytes.NewBuffer([]byte{})
	bufferErr := bytes.NewBuffer([]byte{})
	err = util.OnErrorRetry(maxTries, func(err error) bool {
		return strings.Contains(err.Error(), "error dialing backend: EOF")
	}, func() error {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdout: buffer,
			Stderr: bufferErr,
			Tty:    false,
		})
		if err != nil {
			if strings.Contains(err.Error(), "error dialing backend: EOF") {
				logger.Error(err, fmt.Sprintf("Error executing '%s' in pod %s. Trying again.", command, pod.Name))
			}
		}
		return err
	})
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("error streaming command to pod; out: '%s': errOut: '%s': %w", buffer, bufferErr, err),
			resource:    pod,
		}
	}

	return buffer, nil
}

func (ce *commandExecutor) waitForPodToHaveExpectedStatus(pod *corev1.Pod, expected PodStatus) error {
	var err error
	err = util.OnErrorRetry(maxTries, func(err error) bool {
		_, ok := err.(*unexpectedStatusError)
		return ok
	}, func() error {
		return podHasStatus(pod, expected)
	})

	return err
}

type unexpectedStatusError struct {
	expected PodStatus
}

func (u *unexpectedStatusError) Error() string {
	return fmt.Sprintf("expected status %s not fulfilled", u.expected)
}

func podHasStatus(pod *corev1.Pod, expected PodStatus) error {
	switch expected {
	case ContainersStarted:
		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}
	case PodReady:
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.ContainersReady && condition.Status == corev1.ConditionTrue {
				return nil
			}
		}
	default:
		return fmt.Errorf("unsupported pod status: %s", expected)
	}

	return &unexpectedStatusError{expected: expected}
}

func (ce *commandExecutor) getCreateExecRequest(pod *corev1.Pod, command *ShellCommand) *rest.Request {
	return ce.CoreV1RestClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: append([]string{command.Command}, command.Args...),
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			// Note: if the TTY is set to true shell commands may emit ANSI codes into the stdout
			TTY: false,
		}, scheme.ParameterCodec)
}
