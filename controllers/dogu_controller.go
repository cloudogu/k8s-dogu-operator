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
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Operation int

const (
	Install Operation = iota
	Upgrade
	Delete
	Ignore
)

func (o Operation) toString() string {
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

// DoguReconciler reconciles a Dogu object
type DoguReconciler struct {
	client.Client
	scheme             *runtime.Scheme
	doguManager        Manager
	DoguRequeueHandler RequeueHandler
}

// Manager abstracts the simple dogu operations in a k8s CES.
type Manager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// RequeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type RequeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error) (ctrl.Result, error)
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, scheme *runtime.Scheme, doguManager Manager) *DoguReconciler {
	doguStatusReporter := resource.NewDoguStatusReporter(client)
	doguRequeueHandler := NewDoguRequeueHandler(client, doguStatusReporter)

	return &DoguReconciler{
		Client:             client,
		scheme:             scheme,
		doguManager:        doguManager,
		DoguRequeueHandler: doguRequeueHandler,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DoguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &k8sv1.Dogu{}
	err := r.Get(ctx, req.NamespacedName, doguResource)
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
		installError := r.doguManager.Install(ctx, doguResource)
		if installError != nil {
			doguResource.Status.Status = k8sv1.DoguStatusNotInstalled
		}
		contextMessageOnError := fmt.Sprintf("failed to install dogu %s", doguResource.Name)
		return r.DoguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, installError)
	case Upgrade:
		return ctrl.Result{}, nil
	case Delete:
		err := r.doguManager.Delete(ctx, doguResource)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete dogu: %w", err)
		}
		return ctrl.Result{}, nil
	case Ignore:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func evaluateRequiredOperation(doguResource *k8sv1.Dogu, logger logr.Logger) (Operation, error) {
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

// SetupWithManager sets up the controller with the Manager.
func (r *DoguReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
