package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"

	appsv1 "k8s.io/api/apps/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	deploy "github.com/cloudogu/k8s-dogu-operator/controllers/deployment"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

const legacyDoguLabel = "dogu"

// DeploymentReconciler watches every Deployment object in the cluster and writes the state of dogus into their respective custom resources.
type DeploymentReconciler struct {
	deployClient            appsv1client.DeploymentInterface
	availabilityChecker     cloudogu.DeploymentAvailabilityChecker
	doguHealthStatusUpdater cloudogu.DoguHealthStatusUpdater
}

func NewDeploymentReconciler(deployClient appsv1client.DeploymentInterface, doguClient ecoSystem.DoguInterface, recorder record.EventRecorder) *DeploymentReconciler {
	return &DeploymentReconciler{
		deployClient:            deployClient,
		availabilityChecker:     &deploy.AvailabilityChecker{},
		doguHealthStatusUpdater: health.NewDoguStatusUpdater(doguClient, recorder),
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (dr *DeploymentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	deployment, err := dr.deployClient.Get(ctx, request.Name, metav1api.GetOptions{})
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
	logger.Info(fmt.Sprintf("Found dogu deployment: [%s]", deployment.Name))

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
	return dr.doguHealthStatusUpdater.UpdateStatus(ctx, doguDeployment.Name, doguAvailable)
}

// SetupWithManager sets up the controller with the Manager.
func (dr *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(dr)
}
