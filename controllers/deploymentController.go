package controllers

import (
	"context"
	"fmt"
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

	deployment, err := dr.getDeployment(ctx, request)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get deployment %q: %s", request.NamespacedName, err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !hasDoguLabel(deployment) {
		// ignore non dogu deployments
		return ctrl.Result{}, nil
	}
	logger.Info(fmt.Sprintf("Found dogu deployment: [%s]", deployment.Name))

	err = dr.updateDoguHealth(ctx, deployment)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to update dogu health for deployment %q: %s", request.NamespacedName, err))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (dr *DeploymentReconciler) getDeployment(ctx context.Context, req ctrl.Request) (*appsv1.Deployment, error) {
	deployment, err := dr.deployClient.Get(ctx, req.Name, metav1api.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	return deployment, nil
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
