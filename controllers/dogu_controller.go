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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DoguReconciler reconciles a Dogu object
type DoguReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogus/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DoguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	var doguResource k8sv1.Dogu
	err := r.Get(ctx, req.NamespacedName, &doguResource)
	if err != nil {
		logger.Error(err, "failed to get doguResource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("found doguResource in state: %+v", doguResource))

	// Create deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.Name,
		Namespace: doguResource.Namespace,
	}}

	result, err := ctrl.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		modifyDeploymentForDogu(deployment, doguResource)
		return ctrl.SetControllerReference(&doguResource, deployment, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info(fmt.Sprintf("createOrUpdate deployment result: %+v", result))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		Complete(r)
}

func modifyDeploymentForDogu(deployment *appsv1.Deployment, doguResource k8sv1.Dogu) {
	labels := map[string]string{"dogu": doguResource.Name}
	deployment.ObjectMeta.Labels = labels
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Strategy: appsv1.DeploymentStrategy{
			Type: "Recreate",
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Hostname: doguResource.Name,
				Containers: []corev1.Container{{Name: doguResource.Name,
					//Image: dogu.Image + ":" + dogu.Version,
					Image:           "dummy",
					ImagePullPolicy: corev1.PullIfNotPresent}},
			},
		},
	}
}
