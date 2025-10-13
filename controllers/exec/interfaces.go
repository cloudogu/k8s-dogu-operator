package exec

import (
	"bytes"
	"context"
	"io"

	"k8s.io/client-go/tools/remotecommand"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"

	"github.com/cloudogu/cesapp-lib/core"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

const (
	// ContainersStarted means that all containers of a pod were started.
	ContainersStarted PodStatusForExec = "started"
	// PodReady means that the readiness probe of the pod has succeeded.
	PodReady PodStatusForExec = "ready"
)

// ExecPodFactory provides methods for instantiating and removing an intermediate pod based on a Dogu container image.
type ExecPodFactory interface {
	// CreateBlocking adds a new exec pod to the cluster and waits for its creation.
	// Deprecated, as we don't want our code to block.
	CreateBlocking(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error
	// Create adds a new exec pod to the cluster.
	Create(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error
	Exists(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) bool
	CheckReady(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error
	// Delete deletes the exec pod from the cluster.
	Delete(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) error
	// Exec runs the provided command in this execPodFactory
	Exec(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu, cmd ShellCommand) (out *bytes.Buffer, err error)
}

// CommandExecutor is used to execute commands in pods and dogus
type CommandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, resource *k8sv2.Dogu, command ShellCommand) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command ShellCommand) (*bytes.Buffer, error)
}

// ShellCommand represents a command that can be executed in the shell of a container.
type ShellCommand interface {
	// CommandWithArgs returns the commands and its arguments in a way suitable for execution.
	CommandWithArgs() []string
	// Stdin returns the appropriate reader for standard input.
	Stdin() io.Reader
}

// PodStatusForExec describes a state in the lifecycle of a pod.
type PodStatusForExec string

// FileExtractor provides functionality to get the contents of files from an exec pod.
type FileExtractor interface {
	// ExtractK8sResourcesFromExecPod copies files from an exec pod into map of strings.
	ExtractK8sResourcesFromExecPod(ctx context.Context, doguResource *k8sv2.Dogu, dogu *core.Dogu) (map[string]string, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type remoteExecutor interface {
	remotecommand.Executor
}

//nolint:unused
//goland:noinspection GoUnusedType
type k8sClient interface {
	client.Client
}
