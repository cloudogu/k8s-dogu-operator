package controllers

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//nolint:unused
//goland:noinspection GoUnusedType
type appsV1Interface interface {
	appsv1client.AppsV1Interface
}

type ClientSet interface {
	kubernetes.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	v1.CoreV1Interface
}

type K8sClient interface {
	client.Client
}

type podInterface interface {
	v1.PodInterface
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
}

type doguInterface interface {
	doguClient.DoguInterface
}

type eventRecorder interface {
	record.EventRecorder
}

type DoguReconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager, externalEvents <-chan event.TypedGenericEvent[*v2.Dogu]) error
}

type GenericReconciler interface {
	reconcile.Reconciler
	SetupWithManager(mgr ctrl.Manager) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguRestartInterface interface {
	doguClient.DoguRestartInterface
}

type DoguUsecase interface {
	HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error)
}

type DoguRestartManager interface {
	RestartAllDogus(ctx context.Context) error
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}

type configMapInterface interface {
	v1.ConfigMapInterface
}
