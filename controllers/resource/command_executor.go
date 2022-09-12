package resource

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
)

// resourceStateError is returned when a specific resource (pod/dogu) is not ready yet.
type resourceStateError struct {
	sourceError error
	resource    metav1.Object
}

// Report returns the error in string representation
func (e *resourceStateError) Error() string {
	return fmt.Sprintf("resource is not ready: %v, source error: %s", e.resource.GetName(), e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *resourceStateError) Requeue() bool {
	return true
}

// exposedCommandExecutor is the unit to execute exposed commands in a dogu
type exposedCommandExecutor struct {
	Client                 kubernetes.Interface `json:"client"`
	CoreV1RestClient       rest.Interface       `json:"coreV1RestClient"`
	CommandExecutorCreator func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)
}

// NewCommandExecutor creates a new instance of NewCommandExecutor
func NewCommandExecutor(client kubernetes.Interface, coreV1RestClient rest.Interface) *exposedCommandExecutor {
	return &exposedCommandExecutor{
		Client:                 client,
		CoreV1RestClient:       coreV1RestClient,
		CommandExecutorCreator: remotecommand.NewSPDYExecutor,
	}
}

func (ce *exposedCommandExecutor) allContainersReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.ContainersReady {
			return true
		}
	}
	return false
}

// ExecCommand execs an exposed command in the first found pod of a dogu
func (ce *exposedCommandExecutor) ExecCommand(ctx context.Context, targetDogu string, namespace string,
	command *core.ExposedCommand, params []string) (*bytes.Buffer, error) {
	pod, err := ce.getTargetDoguPod(ctx, targetDogu, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod for dogu %s: %w", targetDogu, err)
	}

	if !ce.allContainersReady(pod) {
		return nil, &resourceStateError{
			sourceError: fmt.Errorf("can't execute command in pod with status %v", pod.Status),
			resource:    pod,
		}
	}

	req := ce.getCreateExecRequest(pod, namespace, command, params)
	exec, err := ce.CommandExecutorCreator(ctrl.GetConfigOrDie(), "POST", req.URL())
	if err != nil {
		return nil, &resourceStateError{
			sourceError: fmt.Errorf("failed to create new spdy executor: %w", err),
			resource:    pod,
		}
	}

	buffer := bytes.NewBuffer([]byte{})
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buffer,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return nil, &resourceStateError{
			sourceError: err,
			resource:    pod,
		}
	}

	return buffer, nil
}

func (ce *exposedCommandExecutor) getCreateExecRequest(pod *corev1.Pod, namespace string,
	createCommand *core.ExposedCommand, params []string) *rest.Request {
	return ce.CoreV1RestClient.Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: append([]string{createCommand.Command}, params...),
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
}

func (ce *exposedCommandExecutor) getTargetDoguPod(ctx context.Context, targetDogu string, namespace string) (*corev1.Pod, error) {
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
