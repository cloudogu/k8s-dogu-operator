package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	appsv1 "k8s.io/api/apps/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

const legacyDoguLabel = "dogu"

// DeploymentReconciler watches every Deployment object in the cluster and writes the state of dogus into their respective custom resources.
type DeploymentReconciler struct {
	k8sClientSet            thirdParty.ClientSet
	availabilityChecker     cloudogu.DeploymentAvailabilityChecker
	doguHealthStatusUpdater cloudogu.DoguHealthStatusUpdater
}

func NewDeploymentReconciler(k8sClientSet thirdParty.ClientSet, ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface, recorder record.EventRecorder) *DeploymentReconciler {
	return &DeploymentReconciler{
		k8sClientSet:            k8sClientSet,
		availabilityChecker:     &health.AvailabilityChecker{},
		doguHealthStatusUpdater: health.NewDoguStatusUpdater(ecosystemClient, recorder),
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (dr *DeploymentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	deployment, err := dr.k8sClientSet.
		AppsV1().Deployments(request.Namespace).
		Get(ctx, request.Name, metav1api.GetOptions{})
	if err != nil {
		return finishOrRequeue(logger,
			client.IgnoreNotFound(
				fmt.Errorf("failed to get deployment %q: %w", request.NamespacedName, err),
			),
		)
	}

	if !hasDoguLabel(deployment) {
		// ignore non dogu deployments
		return finishOperation()
	}
	logger.Info(fmt.Sprintf("Found dogu deployment %q", deployment.Name))

	err = dr.updateDoguHealth(ctx, deployment)
	if err != nil {
		return finishOrRequeue(logger, fmt.Errorf("failed to update dogu health for deployment %q: %w", request.NamespacedName, err))
	}

	return finishOperation()
}

func finishOrRequeue(logger logr.Logger, err error) (ctrl.Result, error) {
	if err != nil {
		logger.Error(err, "reconcile failed")
	}

	return ctrl.Result{}, err
}

func hasDoguLabel(deployment client.Object) bool {
	for label := range deployment.GetLabels() {
		if label == legacyDoguLabel || label == doguv1.DoguLabelName {
			return true
		}
	}

	return false
}

func (dr *DeploymentReconciler) updateDoguHealth(ctx context.Context, doguDeployment *appsv1.Deployment) error {
	doguAvailable := dr.availabilityChecker.IsAvailable(doguDeployment)
	log.FromContext(ctx).Info(fmt.Sprintf("dogu deployment %q is %s", doguDeployment.Name, (map[bool]string{true: "available", false: "unavailable"})[doguAvailable]))
	return dr.doguHealthStatusUpdater.UpdateStatus(ctx,
		types.NamespacedName{Name: doguDeployment.Name, Namespace: doguDeployment.Namespace},
		doguAvailable)
}

// SetupWithManager sets up the controller with the Manager.
func (dr *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(dr)
}