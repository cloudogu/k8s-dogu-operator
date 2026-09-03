package controllers

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// basically a reimplementation of manager.Manager, but the mocks generate illegal files out of that interface
//
//nolint:unused
//goland:noinspection GoUnusedType
type ctrlManager interface {
	manager.Manager
}

type ClientSet interface {
	kubernetes.Interface
}

type K8sClient interface {
	client.Client
}

type doguInterface interface {
	doguv2.DoguInterface
}

type eventRecorder interface {
	record.EventRecorder
}

type GenericReconciler interface {
	reconcile.Reconciler
	setupWithManager(mgr ctrlManager) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguRestartInterface interface {
	doguv2.DoguRestartInterface
}

type DoguUsecase interface {
	HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error)
}

// RequeueHandlerV2 abstracts the process to decide whether a requeue process should be done based on received errors.
type RequeueHandlerV2 interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, doguResource *v2.Dogu, err error, reqTime time.Duration) (result ctrl.Result, requeueErr error)
}

// RequeueHandlerV3 abstracts the process to decide whether a requeue process should be done based on received errors.
type RequeueHandlerV3 interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, doguResource *v3beta1.Dogu, name types.NamespacedName) (result ctrl.Result, requeueErr error)
}

type DoguInstallOrChangeUseCase interface {
	DoguUsecase
}
type DoguDeleteUseCase interface {
	DoguUsecase
}

//nolint:unused
//goland:noinspection GoUnusedType
type WebhookServer interface {
	webhook.Server
}
