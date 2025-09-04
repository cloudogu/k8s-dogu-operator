package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ReasonDoguNotHealthy = "DoguIsNotHealthy"
	ReasonDoguHealthy    = "DoguIsHealthy"
)

type HealthCheckStep struct {
	client                  client.Client
	availabilityChecker     health.DeploymentAvailabilityChecker
	doguHealthStatusUpdater health.DoguHealthStatusUpdater
	doguFetcher             localDoguFetcher
	doguInterface           doguInterface
}

func NewHealthCheckStep(client client.Client, availabilityChecker *health.AvailabilityChecker,
	doguHealthStatusUpdater health.DoguHealthStatusUpdater, mgrSet *util.ManagerSet, namespace string) *HealthCheckStep {
	return &HealthCheckStep{
		client:                  client,
		availabilityChecker:     availabilityChecker,
		doguHealthStatusUpdater: doguHealthStatusUpdater,
		doguFetcher:             mgrSet.LocalDoguFetcher,
		doguInterface:           mgrSet.EcosystemClient.Dogus(namespace),
	}
}

func (hcs *HealthCheckStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, hcs.client)
	if err != nil {
		if errors.IsNotFound(err) {
			steps.Continue()
		}
		return steps.RequeueWithError(err)
	}
	err = hcs.updateDoguHealth(ctx, deployment, doguResource)
	if err != nil {
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
	message := ""
	if doguAvailable {
		status = metav1.ConditionTrue
		reason = ReasonDoguHealthy
	}
	log.FromContext(ctx).Info(fmt.Sprintf("dogu deployment %q is %s", doguDeployment.Name, (map[bool]string{true: "available", false: "unavailable"})[doguAvailable]))
	condition := metav1.Condition{
		Type:               doguv2.ConditionHealthy,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	doguResource, err = hcs.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err = hcs.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
