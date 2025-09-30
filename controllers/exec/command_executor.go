package exec

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ShellCommand represents all necessary arguments to execute a command inside a container.
type shellCommand struct {
	// command states the actual executable that is supposed to be executed in the container.
	command string
	// args contains any parameters, switches etc. that the command needs to run properly.
	args []string
	// stdin contains standard input for the command, input that could for example be piped in a CLI environment.
	stdin io.Reader
}

func (sc *shellCommand) Stdin() io.Reader {
	return sc.stdin
}

func (sc *shellCommand) CommandWithArgs() []string {
	return append([]string{sc.command}, sc.args...)
}

func (sc *shellCommand) String() string {
	result := []string{sc.command}
	return strings.Join(append(result, sc.args...), " ")
}

// NewShellCommand creates a new ShellCommand. While the command is mandatory, there can be zero to n command arguments.
func NewShellCommand(command string, args ...string) *shellCommand {
	return &shellCommand{command: command, args: args}
}

func NewShellCommandWithStdin(stdin io.Reader, command string, args ...string) *shellCommand {
	return &shellCommand{command: command, args: args, stdin: stdin}
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
var (
	maxTries  = 20
	waitLimit = time.Minute * 30
)

// commandExecutor is the unit to execute commands in a dogu
type defaultCommandExecutor struct {
	client                 client.Client
	restConfig             *rest.Config
	clientSet              kubernetes.Interface
	coreV1RestClient       rest.Interface
	commandExecutorCreator func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)
}

// NewCommandExecutor creates a new instance of NewCommandExecutor
func NewCommandExecutor(cli client.Client, restConfig *rest.Config, clientSet kubernetes.Interface, coreV1RestClient rest.Interface) CommandExecutor {
	return &defaultCommandExecutor{
		client:     cli,
		restConfig: restConfig,
		clientSet:  clientSet,
		// the rest clientSet COULD be generated from the clientSet but makes harder to test, so we source it additionally
		coreV1RestClient:       coreV1RestClient,
		commandExecutorCreator: remotecommand.NewSPDYExecutor,
	}
}

// ExecCommandForDogu execs a command in the first found pod of a dogu. This method executes a command on a dogu pod
// that can be selected by a K8s label.
func (ce *defaultCommandExecutor) ExecCommandForDogu(ctx context.Context, resource *v2.Dogu, command ShellCommand) (*bytes.Buffer, error) {
	updatedDogu := &v2.Dogu{}
	err := ce.client.Get(ctx, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, updatedDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu %q: %w", resource.Name, err)
	}

	if conditions.IsFalse(updatedDogu, v2.ConditionHealthy) {
		return nil, fmt.Errorf("dogu %q is not available", resource.Name)
	}

	pod, err := v2.GetPodForLabels(ctx, ce.client, updatedDogu.GetPodLabelsWithStatusVersion())
	if err != nil {
		return nil, fmt.Errorf("failed to get pod from dogu %q: %w", resource.Name, err)
	}

	return ce.ExecCommandForPod(ctx, pod, command)
}

// ExecCommandForPod execs a command in a given pod. This method executes a command on an arbitrary pod that can be
// identified by its pod name.
func (ce *defaultCommandExecutor) ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command ShellCommand) (*bytes.Buffer, error) {
	req := ce.getCreateExecRequest(pod, command)
	exec, err := ce.commandExecutorCreator(ctrl.GetConfigOrDie(), "POST", req.URL())
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("failed to create new spdy executor: %w", err),
			resource:    pod,
		}
	}

	return ce.streamCommandToPod(ctx, exec, command, pod)
}

func (ce *defaultCommandExecutor) streamCommandToPod(
	ctx context.Context,
	exec remotecommand.Executor,
	command ShellCommand,
	pod *corev1.Pod,
) (*bytes.Buffer, error) {
	stdin := command.Stdin()
	buffer := bytes.NewBuffer([]byte{})
	bufferErr := bytes.NewBuffer([]byte{})
	err := exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: buffer,
		Stderr: bufferErr,
		Tty:    false,
	})
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("error streaming command to pod; out: '%s': errOut: '%s': %w", buffer, bufferErr, err),
			resource:    pod,
		}
	}

	return buffer, nil
}

func (ce *defaultCommandExecutor) getCreateExecRequest(pod *corev1.Pod, command ShellCommand) *rest.Request {
	hasStdin := command.Stdin() != nil
	return ce.coreV1RestClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: ce.getDefaultContainer(pod),
			Command:   command.CommandWithArgs(),
			Stdin:     hasStdin,
			Stdout:    true,
			Stderr:    true,
			// Note: if the TTY is set to true shell commands may emit ANSI codes into the stdout
			TTY: false,
		}, scheme.ParameterCodec)
}

func (ce *defaultCommandExecutor) getDefaultContainer(pod *corev1.Pod) string {
	if container, ok := pod.Annotations["kubectl.kubernetes.io/default-container"]; ok {
		return container
	}
	return ""
}
