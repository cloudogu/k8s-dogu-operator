package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReasonDoguNotHealthy = "DoguIsNotHealthy"
	ReasonDoguHealthy    = "DoguIsHealthy"
)

type HealthCheckStep struct {
	client                  k8sClient
	availabilityChecker     deploymentAvailabilityChecker
	doguHealthStatusUpdater doguHealthStatusUpdater
	doguFetcher             localDoguFetcher
	doguInterface           doguInterface
}

func NewHealthCheckStep(client client.Client, availabilityChecker health.DeploymentAvailabilityChecker,
	doguHealthStatusUpdater health.DoguHealthStatusUpdater, fetcher cesregistry.LocalDoguFetcher, doguInterface doguClient.DoguInterface) *HealthCheckStep {
	return &HealthCheckStep{
		client:                  client,
		availabilityChecker:     availabilityChecker,
		doguHealthStatusUpdater: doguHealthStatusUpdater,
		doguFetcher:             fetcher,
		doguInterface:           doguInterface,
	}
}

func (hcs *HealthCheckStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, hcs.client)
	if err != nil {
		if errors.IsNotFound(err) {
			return steps.Continue()
		}
		return steps.RequeueWithError(err)
	}

	err = hcs.updateDoguHealth(ctx, deployment, doguResource)
	if err != nil {
		if cloudoguerrors.IsNotFoundError(err) {
			return steps.Continue()
		}
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}

func (hcs *HealthCheckStep) updateDoguHealth(ctx context.Context, doguDeployment *appsv1.Deployment, doguResource *doguv2.Dogu) error {
	doguAvailable := hcs.availabilityChecker.IsAvailable(doguDeployment)
	doguJson, err := hcs.doguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguDeployment.Name))
	if err != nil {
		return fmt.Errorf("failed to get current dogu json to update health state configMap: %w", err)
	}
	err = hcs.doguHealthStatusUpdater.UpdateHealthConfigMap(ctx, doguDeployment, doguJson)
	if err != nil {
		return fmt.Errorf("failed to update health state configMap: %w", err)
	}

	status := metav1.ConditionFalse
	reason := ReasonDoguNotHealthy
	message := "Not all replicas are available"
	desiredHealthStatus := doguv2.UnavailableHealthStatus
	if doguAvailable {
		status = metav1.ConditionTrue
		reason = ReasonDoguHealthy
		message = "All replicas are available"
		desiredHealthStatus = doguv2.AvailableHealthStatus
	}

	condition := metav1.Condition{
		Type:               doguv2.ConditionHealthy,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now().Rfc3339Copy(),
	}

	doguResource.Status.Health = desiredHealthStatus
	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err = hcs.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
