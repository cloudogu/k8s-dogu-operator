// Package external implements mocks that implement 3rd party interfaces, t. i. interfaces which we do not control.
// In order to avoid package dependency cycles these mock implementations reside in this package.
package thirdParty

import (
	"context"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/go-logr/logr"

	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/cloudogu/cesapp-lib/remote"
)

type K8sClient interface {
	client.Client
}

type K8sSubResourceWriter interface {
	client.SubResourceWriter
}

type RemoteExecutor interface {
	remotecommand.Executor
}

type EventRecorder interface {
	record.EventRecorder
}

type LogSink interface {
	logr.LogSink
}

type ControllerManager interface {
	manager.Manager
}

// RemoteRegistry is able to manage the remote dogu registry.
type RemoteRegistry interface {
	remote.Registry
}

// LocalDoguRegistry abstracts accessing various backends for reading and writing dogu specs (dogu.json).
type LocalDoguRegistry interface {
	dogu.LocalRegistry
}

type DeploymentInterface interface {
	appsv1client.DeploymentInterface
}

type ConfigMapInterface interface {
	v1.ConfigMapInterface
}

type PodInterface interface {
	v1.PodInterface
}

type AppsV1Interface interface {
	appsv1client.AppsV1Interface
}

type CoreV1Interface interface {
	v1.CoreV1Interface
}

type ClientSet interface {
	kubernetes.Interface
}

type ConfigMapClient interface {
	repository.ConfigMapClient
}

type GlobalConfigRepository interface {
	Get(ctx context.Context) (config.GlobalConfig, error)
	Create(ctx context.Context, globalConfig config.GlobalConfig) (config.GlobalConfig, error)
	Update(ctx context.Context, globalConfig config.GlobalConfig) (config.GlobalConfig, error)
	SaveOrMerge(ctx context.Context, globalConfig config.GlobalConfig) (config.GlobalConfig, error)
	Delete(ctx context.Context) error
	Watch(ctx context.Context, filters ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error)
}

type DoguConfigRepository interface {
	Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error)
	Create(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Delete(ctx context.Context, name config.SimpleDoguName) error
	Watch(ctx context.Context, dName config.SimpleDoguName, filters ...config.WatchFilter) (<-chan repository.DoguConfigWatchResult, error)
}
