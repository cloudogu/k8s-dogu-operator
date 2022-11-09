package exec

import (
	"bytes"
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	Exec(ctx context.Context, cmd *ShellCommand) (out string, err error)
}

type suffixGenerator interface {
	// String returns a random suffix string with the given length
	String(length int) string
}

// commandExecutor is used to execute commands in pods and dogus
type commandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, resource *k8sv1.Dogu, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command *ShellCommand, expectedStatus PodStatus) (*bytes.Buffer, error)
}
