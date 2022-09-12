/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

type operation int

const (
	InstallEventReason        = "Installation"
	ErrorOnInstallEventReason = "ErrInstallation"
	DeinstallEventReason      = "Deinstallation"
	ErrorDeinstallEventReason = "ErrDeinstallation"
	SupportEventReason        = "Support"
	ErrorOnSupportEventReason = "ErrSupport"
	RequeueEventReason        = "Requeue"
	ErrorOnRequeueEventReason = "ErrRequeue"
)

const (
	Install operation = iota
	Upgrade
	Delete
	Ignore
)

func (o operation) toString() string {
	switch o {
	case Install:
		return "Install"
	case Upgrade:
		return "Upgrade"
	case Delete:
		return "Delete"
	default:
		return "Ignore"
	}
}

// doguReconciler reconciles a Dogu object
type doguReconciler struct {
	client             client.Client
	scheme             *runtime.Scheme
	doguManager        manager
	doguRequeueHandler requeueHandler
	recorder           record.EventRecorder
}

// manager abstracts the simple dogu operations in a k8s CES.
type manager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
	// HandleSupportFlag handles the support flag in the dogu spec.
	HandleSupportFlag(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error)
}

// requeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type requeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error, onRequeue func(dogu *k8sv1.Dogu)) (ctrl.Result, error)
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, scheme *runtime.Scheme, doguManager manager, eventRecorder record.EventRecorder, namespace string) (*doguReconciler, error) {
	doguRequeueHandler, err := NewDoguRequeueHandler(client, eventRecorder, namespace)
	if err != nil {
		return nil, err
	}

	return &doguReconciler{
		client:             client,
		scheme:             scheme,
		doguManager:        doguManager,
		doguRequeueHandler: doguRequeueHandler,
		recorder:           eventRecorder,
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims tomal
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *doguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &k8sv1.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been found", doguResource.Namespace, doguResource.Name))

	requiredOperation, err := evaluateRequiredOperation(doguResource, logger)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to evaluate required operation: %w", err)
	}

	logger.Info(fmt.Sprintf("Required operation for Dogu %s/%s is: %s", doguResource.Namespace, doguResource.Name, requiredOperation.toString()))

	switch requiredOperation {
	case Install:
		return r.handleInstallOperation(ctx, doguResource)
	case Upgrade:
		supportResult, err := r.handleSupportFlag(ctx, doguResource)
		if supportResult != nil {
			return *supportResult, err
		}
		return ctrl.Result{}, nil
	case Delete:
		return r.handleDeleteOperation(ctx, doguResource)
	case Ignore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func (r *doguReconciler) handleSupportFlag(ctx context.Context, doguResource *k8sv1.Dogu) (*reconcile.Result, error) {
	logger := log.FromContext(ctx)
	// Only recognise support mode if dogu is installed.
	if doguResource.Status.Status == k8sv1.DoguStatusInstalled {
		// Handle support mode flag and detect if the support mode changed.
		logger.Info(fmt.Sprintf("Handling support flag for dogu %s", doguResource.Name))
		supportModeChanged, err := r.doguManager.HandleSupportFlag(ctx, doguResource)
		if err != nil {
			printError := strings.ReplaceAll(err.Error(), "\n", "")
			r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnSupportEventReason, "Handling of support mode failed.", printError)
			return &ctrl.Result{}, fmt.Errorf("failed to handle support mode: %w", err)
		}

		// Do not care about other operations if the mode has changed. Data change with support won't and shouldn't be processed.
		logger.Info(fmt.Sprintf("Check if support mode changed for dogu %s", doguResource.Name))
		logger.Info(fmt.Sprintf("Changed %t", supportModeChanged))
		if supportModeChanged {
			r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode has changed. Ignoring other events.")
			return &ctrl.Result{}, nil
		}

		// Do not care about other operations if the support mode is currently active.
		if doguResource.Spec.SupportMode {
			logger.Info(fmt.Sprintf("Support mode is currently active for dogu %s", doguResource.Name))
			r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode is active. Ignoring other events.")
			return &ctrl.Result{}, nil
		}
	}

	return nil, nil
}

func (r *doguReconciler) handleInstallOperation(ctx context.Context, doguResource *k8sv1.Dogu) (reconcile.Result, error) {
	installError := r.doguManager.Install(ctx, doguResource)
	contextMessageOnError := fmt.Sprintf("failed to install dogu %s", doguResource.Name)

	if installError == nil {
		r.recorder.Event(doguResource, v1.EventTypeNormal, InstallEventReason, "Installation successful.")
		return ctrl.Result{}, nil
	}

	printError := strings.Replace(installError.Error(), "\n", "", -1)
	r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnInstallEventReason, "Installation failed. Reason: %s.", printError)

	result, err := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, installError, func(dogu *k8sv1.Dogu) {
		doguResource.Status.Status = k8sv1.DoguStatusNotInstalled
	})
	if err != nil {
		r.recorder.Event(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason, "Failed to requeue the installation.")
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", err)
	}

	return result, nil
}

func (r *doguReconciler) handleDeleteOperation(ctx context.Context, doguResource *k8sv1.Dogu) (reconcile.Result, error) {
	deleteError := r.doguManager.Delete(ctx, doguResource)
	contextMessageOnError := fmt.Sprintf("failed to delete dogu %s", doguResource.Name)

	if deleteError == nil {
		r.recorder.Event(doguResource, v1.EventTypeNormal, DeinstallEventReason, "Deinstallation successful.")
		return ctrl.Result{}, nil
	}

	printError := strings.Replace(deleteError.Error(), "\n", "", -1)
	r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorDeinstallEventReason, "Deinstallation failed. Reason: %s.", printError)

	result, err := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, deleteError, func(dogu *k8sv1.Dogu) {
		doguResource.Status.Status = k8sv1.DoguStatusInstalled
	})
	if err != nil {
		r.recorder.Event(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason, "Failed to requeue the deinstallation.")
		return ctrl.Result{}, fmt.Errorf("failed to handle requeue: %w", err)
	}

	return result, nil
}

func evaluateRequiredOperation(doguResource *k8sv1.Dogu, logger logr.Logger) (operation, error) {
	if !doguResource.DeletionTimestamp.IsZero() {
		return Delete, nil
	}

	switch doguResource.Status.Status {
	case k8sv1.DoguStatusNotInstalled:
		return Install, nil
	case k8sv1.DoguStatusInstalled:
		// Checking if the resource spec field has changed is unnecessary because we
		// use a predicate to filter update events where specs don't change
		return Upgrade, nil
	case k8sv1.DoguStatusInstalling:
		return Ignore, nil
	case k8sv1.DoguStatusDeleting:
		return Ignore, nil
	default:
		logger.Info(fmt.Sprintf("Found unknown operation for dogu status: %s", doguResource.Status.Status))
		return Ignore, nil
	}
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		// Since we don't want to process dogus with same spec we use a generation change predicate
		// as a filter to reduce the reconcile calls.
		// The predicate implements a function that will be invoked of every update event that
		// the k8s api will fire. On writing the objects spec field the k8s api
		// increments the generation field. The function compares this field from the old
		// and new dogu resource. If they are equal the reconcile loop will not be called.
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
