package controllers

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const legacyDoguLabel = "dogu"

// DeploymentReconciler watches every Deployment object in the cluster and writes the state of dogus into their respective custom resources.
type DeploymentReconciler struct {
	k8sClientSet            ClientSet
	availabilityChecker     health.DeploymentAvailabilityChecker
	doguHealthStatusUpdater health.DoguHealthStatusUpdater
	doguFetcher             localDoguFetcher
}

func NewDeploymentReconciler(k8sClientSet ClientSet, availabilityChecker *health.AvailabilityChecker,
	doguHealthStatusUpdater health.DoguHealthStatusUpdater, doguFetcher localDoguFetcher) *DeploymentReconciler {
	return &DeploymentReconciler{
		k8sClientSet:            k8sClientSet,
		availabilityChecker:     availabilityChecker,
		doguHealthStatusUpdater: doguHealthStatusUpdater,
		doguFetcher:             doguFetcher,
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

func hasDoguLabel(object client.Object) bool {
	for label := range object.GetLabels() {
		if label == legacyDoguLabel || label == doguv2.DoguLabelName {
			return true
		}
	}

	return false
}

func getDoguLabel(object client.Object) string {
	for label, value := range object.GetLabels() {
		if label == legacyDoguLabel || label == doguv2.DoguLabelName {
			return value
		}
	}

	return ""
}

func (dr *DeploymentReconciler) updateDoguHealth(ctx context.Context, doguDeployment *appsv1.Deployment) error {
	doguAvailable := dr.availabilityChecker.IsAvailable(doguDeployment)
	doguJson, err := dr.doguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguDeployment.Name))
	if err != nil {
		return fmt.Errorf("failed to get current dogu json to update health state configMap: %w", err)
	}
	err = dr.doguHealthStatusUpdater.UpdateHealthConfigMap(ctx, doguDeployment, doguJson)
	if err != nil {
		return fmt.Errorf("failed to update health state configMap: %w", err)
	}

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
