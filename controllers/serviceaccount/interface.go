package serviceaccount

import (
	"bytes"
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	"github.com/cloudogu/k8s-registry-lib/config"
	corev1 "k8s.io/api/core/v1"
)

type SensitiveDoguConfigProvider interface {
	GetSensitiveDoguConfig(ctx context.Context, doguName string) (SensitiveDoguConfig, error)
}

type SensitiveDoguConfigSetter interface {
	Set(ctx context.Context, key, value string) error
}

type SensitiveDoguConfigGetter interface {
	Exists(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string) (string, error)
}

type SensitiveDoguConfigDeleter interface {
	DeleteRecursive(ctx context.Context, key string) error
}

type SensitiveDoguConfig interface {
	SensitiveDoguConfigGetter
	SensitiveDoguConfigSetter
	SensitiveDoguConfigDeleter
}

type SensitiveDoguConfigRepository interface {
	Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
}

// ServiceAccountCreator includes functionality to create necessary service accounts for a dogu.
type ServiceAccountCreator interface {
	// CreateAll is used to create all necessary service accounts for the given dogu.
	CreateAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

// ServiceAccountRemover includes functionality to remove existing service accounts for a dogu.
type ServiceAccountRemover interface {
	// RemoveAll is used to remove all existing service accounts for the given dogu.
	RemoveAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName string) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName string) (bool, error)
}

// CommandExecutor is used to execute commands in pods and dogus
type CommandExecutor interface {
	// ExecCommandForDogu executes a command in a dogu.
	ExecCommandForDogu(ctx context.Context, resource *v2.Dogu, command exec.ShellCommand, expected exec.PodStatusForExec) (*bytes.Buffer, error)
	// ExecCommandForPod executes a command in a pod that must not necessarily be a dogu.
	ExecCommandForPod(ctx context.Context, pod *corev1.Pod, command exec.ShellCommand, expected exec.PodStatusForExec) (*bytes.Buffer, error)
}
