package resource

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
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

// stateError is returned when a specific resource (pod/dogu) is not ready yet.
type stateError struct {
	sourceError error
	resource    metav1.Object
}

// Report returns the error in string representation
func (e *stateError) Error() string {
	return fmt.Sprintf("resource is not ready: %v, source error: %s", e.resource.GetName(), e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *stateError) Requeue() bool {
	return true
}

// commandExecutor is the unit to execute commands in a dogu
type commandExecutor struct {
	Client                 kubernetes.Interface `json:"client"`
	CoreV1RestClient       rest.Interface       `json:"coreV1RestClient"`
	CommandExecutorCreator func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)
}

// NewCommandExecutor creates a new instance of NewCommandExecutor
func NewCommandExecutor(client kubernetes.Interface, coreV1RestClient rest.Interface) *commandExecutor {
	return &commandExecutor{
		Client: client,
		// the rest client COULD be generated from the client but makes harder to test, so we source it additionally
		CoreV1RestClient:       coreV1RestClient,
		CommandExecutorCreator: remotecommand.NewSPDYExecutor,
	}
}

func (ce *commandExecutor) allContainersReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.ContainersReady {
			return true
		}
	}
	return false
}

// ExecCommandForDogu execs a command in the first found pod of a dogu. This method executes a command on a dogu pod
// that can be selected by a K8s label.
func (ce *commandExecutor) ExecCommandForDogu(ctx context.Context, targetDogu string, namespace string,
	command *ShellCommand) (*bytes.Buffer, error) {
	pod, err := ce.getTargetDoguPod(ctx, targetDogu, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod for dogu %s: %w", targetDogu, err)
	}

	return ce.execCommand(pod, namespace, command)
}

// ExecCommandForPod execs a command in a given pod. This method executes a command on an arbitrary pod that can be
// identified by its pod name.
func (ce *commandExecutor) ExecCommandForPod(ctx context.Context, targetPod string, namespace string,
	command *ShellCommand) (*bytes.Buffer, error) {
	pod, err := ce.getPodByName(ctx, targetPod, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", targetPod, err)
	}

	return ce.execCommand(pod, namespace, command)
}

func (ce *commandExecutor) execCommand(pod *corev1.Pod, namespace string, command *ShellCommand) (*bytes.Buffer, error) {
	if !ce.allContainersReady(pod) {
		return nil, &stateError{
			sourceError: fmt.Errorf("can't execute command in pod with status %v", pod.Status),
			resource:    pod,
		}
	}

	req := ce.getCreateExecRequest(pod, namespace, command)
	exec, err := ce.CommandExecutorCreator(ctrl.GetConfigOrDie(), "POST", req.URL())
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("failed to create new spdy executor: %w", err),
			resource:    pod,
		}
	}

	buffer := bytes.NewBuffer([]byte{})
	bufferErr := bytes.NewBuffer([]byte{})
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buffer,
		Stderr: bufferErr,
		Tty:    false,
	})
	if err != nil {
		return nil, &stateError{
			sourceError: fmt.Errorf("out: '%s': errOut: '%s': %w", buffer, bufferErr, err),
			resource:    pod,
		}
	}

	return buffer, nil
}

func (ce *commandExecutor) getCreateExecRequest(pod *corev1.Pod, namespace string,
	command *ShellCommand) *rest.Request {
	return ce.CoreV1RestClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: append([]string{command.Command}, command.Args...),
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)
}

func (ce *commandExecutor) getTargetDoguPod(ctx context.Context, targetDogu string, namespace string) (*corev1.Pod, error) {
	// the pod selection must be revised if dogus are horizontally scalable by adding more pods with the same image.
	listOptions := metav1.ListOptions{LabelSelector: "dogu=" + targetDogu}
	pods, err := ce.Client.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("found no pods for dogu %s", targetDogu)
	}

	return &pods.Items[0], nil
}

func (ce *commandExecutor) getPodByName(ctx context.Context, podName string, namespace string) (*corev1.Pod, error) {
	return ce.Client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
}
