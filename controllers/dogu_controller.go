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

type Manager interface {
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

func NewDoguReconciler(client client.Client, scheme *runtime.Scheme, doguManager Manager) *DoguReconciler {
	return &DoguReconciler{
		Client:      client,
		scheme:      scheme,
		doguManager: doguManager,
	}
}

//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus/finalizers,verbs=create;update;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

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
	logger.Info(fmt.Sprintf("found doguResource in state: %+v", doguResource))

	requiredOperation, err := evaluateRequiredOperation(doguResource)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to evaluate required operation: %w", err)
	}

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
		logger.Info(fmt.Sprintf("remove finalizer from doguResource: %+v", doguResource))
		controllerutil.RemoveFinalizer(doguResource, finalizerName)
		err := r.Update(ctx, doguResource)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update dogu: %w", err)
		}
		return ctrl.Result{}, nil
	case Ignore:
		logger.Info(fmt.Sprintf("no operation required for dogu: %+v", doguResource))
		return ctrl.Result{}, nil
	default:
		logger.Info(fmt.Sprintf("unknown operation for dogu: %+v", doguResource))
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

// SetupWithManager sets up the controller with the Manager.
func (r *DoguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		Complete(r)
}
