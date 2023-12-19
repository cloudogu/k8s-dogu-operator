// Package external implements mocks that implement 3rd party interfaces, t. i. interfaces which we do not control.
// In order to avoid package dependency cycles these mock implementations reside in this package.
package thirdParty

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/cloudogu/cesapp-lib/registry"
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

type DoguRegistry interface {
	registry.DoguRegistry
}

// HostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
type HostAliasGenerator interface {
	Generate() (hostAliases []corev1.HostAlias, err error)
}

type ConfigurationContext interface {
	registry.ConfigurationContext
}

type ConfigurationRegistry interface {
	registry.Registry
}

type DeploymentInterface interface {
	appsv1client.DeploymentInterface
}

type AppsV1Interface interface {
	appsv1client.AppsV1Interface
}

type ClientSet interface {
	kubernetes.Interface
}
