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
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const cesLabel = "ces"
const finalizerName = "dogu-finalizer"

type Operation int

const (
	Install Operation = iota
	Upgrade
	Delete
	Ignore
)

// DoguReconciler reconciles a Dogu object
type DoguReconciler struct {
	client.Client
	scheme      *runtime.Scheme
	doguManager Manager
}

// Manager abstracts the simple dogu operations in a k8s ces
type Manager interface {
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, scheme *runtime.Scheme, doguManager Manager) *DoguReconciler {
	return &DoguReconciler{
		Client:      client,
		scheme:      scheme,
		doguManager: doguManager,
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
	logger.Info(fmt.Sprintf("Dogu %s/%s has been found: %+v", doguResource.Namespace, doguResource.Name, doguResource))

	requiredOperation, err := evaluateRequiredOperation(doguResource)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to evaluate required operation: %w", err)
	}
	logger.Info(fmt.Sprintf("Required operation for Dogu %s/%s is: %s", doguResource.Namespace, doguResource.Name, operationToString(requiredOperation)))

	switch requiredOperation {
	case Install:
		err := r.doguManager.Install(ctx, doguResource)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to install dogu: %w", err)
		}
		return ctrl.Result{}, nil
	case Upgrade:
		return ctrl.Result{}, errors.New("not implemented yet")
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

func evaluateRequiredOperation(doguResource *k8sv1.Dogu) (Operation, error) {
	if !isDoguInstalled(doguResource) {
		return Install, nil
	}

	if !doguResource.ObjectMeta.DeletionTimestamp.IsZero() {
		return Delete, nil
	}

	// TODO: Implement upgrade detection

	return Ignore, nil
}

func isDoguInstalled(doguResource *k8sv1.Dogu) bool {
	return controllerutil.ContainsFinalizer(doguResource, finalizerName)
}

func operationToString(operation Operation) string {
	switch operation {
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

// SetupWithManager sets up the controller with the Manager.
func (r *DoguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		Complete(r)
}
