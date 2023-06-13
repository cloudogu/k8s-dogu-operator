package cloudogu

import (
	"bytes"
	"context"
	"io"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/cloudogu/cesapp-lib/core"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
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
	Exec(ctx context.Context, cmd ShellCommand) (out *bytes.Buffer, err error)
}

// ExecPodFactory provides functionality to create ExecPods.
type ExecPodFactory interface {
	// NewExecPod creates a new ExecPod.
	NewExecPod(doguResource *k8sv1.Dogu, dogu *core.Dogu) (ExecPod, error)
}

// CommandExecutor is used to execute commands in pods and dogus
type CommandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, resource *k8sv1.Dogu, command ShellCommand, expected PodStatusForExec) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command ShellCommand, expected PodStatusForExec) (*bytes.Buffer, error)
}

// ShellCommand represents a command that can be executed in the shell of a container.
type ShellCommand interface {
	// CommandWithArgs returns the commands and its arguments in a way suitable for execution.
	CommandWithArgs() []string
	// Stdin returns whether the command has standard input and if so the appropriate reader.
	Stdin() (bool, io.Reader)
}

// PodStatusForExec describes a state in the lifecycle of a pod.
type PodStatusForExec string

const (
	// ContainersStarted means that all containers of a pod were started.
	ContainersStarted PodStatusForExec = "started"
	// PodReady means that the readiness probe of the pod has succeeded.
	PodReady PodStatusForExec = "ready"
)

// FileExtractor provides functionality to get the contents of files from a container.
type FileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod ExecPod) (map[string]string, error)
}

// SuffixGenerator can generate random suffix strings, e.g. for ExecPods.
type SuffixGenerator interface {
	// String returns a random suffix string with the given length
	String(length int) string
}
