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
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/wip"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
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
	DoguStatusReporter StatusReporter
}

// Manager abstracts the simple dogu operations in a k8s CES
type Manager interface {
	// Install installs a dogu resource
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Upgrade upgrades a dogu resource
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	// Delete deletes a dogu resource
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// StatusReporter is responsible to save information in the dogu status via messages
type StatusReporter interface {
	ReportMessage(ctx context.Context, doguResource *k8sv1.Dogu, message string) error
	ReportError(ctx context.Context, doguResource *k8sv1.Dogu, error error) error
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, scheme *runtime.Scheme, doguManager Manager) *DoguReconciler {
	doguStatusReporter := wip.NewDoguStatusReporter(client)

	return &DoguReconciler{
		Client:             client,
		scheme:             scheme,
		doguManager:        doguManager,
		DoguStatusReporter: doguStatusReporter,
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
		err := r.doguManager.Install(ctx, doguResource)
		return r.handleInstallError(ctx, err, doguResource)
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

func (r *DoguReconciler) handleInstallError(ctx context.Context, err error, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err == nil {
		return ctrl.Result{}, nil
	}

	depError := &dependency.ErrorDependencyValidation{}
	// check if err contains a dependency error
	if errors.As(err, &depError) {
		reportError := r.DoguStatusReporter.ReportError(ctx, doguResource, err)
		if reportError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to report error: %w", reportError)
		}

		requeueTime := doguResource.Status.NextRequeue()
		logger.Error(err, fmt.Sprintf("Failed to install dogu %s -> retry dogu installation in %s seconds",
			doguResource.Spec.Name, requeueTime))
		return ctrl.Result{RequeueAfter: requeueTime}, nil
	}

	return ctrl.Result{}, fmt.Errorf("failed to install dogu: %w", err)
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
