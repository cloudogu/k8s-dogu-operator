package migration

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultRequeueAfter = 3 * time.Second
)

type CleanupV2ResourcesStep struct {
	client         client.Client
	resourceFinder ResourceFinder
	eventRecorder  EventRecorder
}

func NewCleanupV2ResourcesStep(k8sClient client.Client, resourceFinder ResourceFinder, eventRecorder EventRecorder) *CleanupV2ResourcesStep {
	return &CleanupV2ResourcesStep{
		client:         k8sClient,
		resourceFinder: resourceFinder,
		eventRecorder:  eventRecorder,
	}
}

func (s *CleanupV2ResourcesStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	resources, err := s.resourceFinder.FindV2Resources(ctx, doguResource.Name)
	if err != nil {
		return stepsv3.RequeueWithError(err, "failed to find v2 resources")
	}

	if len(resources) == 0 {
		return s.updateReadyCondition(ctx, doguResource)
	}

	for _, resource := range resources {
		err = s.client.Delete(ctx, resource)
		if err != nil {
			return stepsv3.RequeueWithError(err, "failed to delete v2 resource")
		}
	}
	return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonMigrationPending, "Resources of v2 dogu are being deleted")
}

func (s *CleanupV2ResourcesStep) updateReadyCondition(ctx context.Context, dogu *v3beta1.Dogu) stepsv3.StepResult {
	if meta.SetStatusCondition(&dogu.Status.Conditions, v1.Condition{
		Type:    v3beta1.ConditionReady,
		Status:  v1.ConditionFalse,
		Reason:  v3beta1.ReasonMigrationPending,
		Message: "Old v2 resources deleted. Migration still pending",
	}) {
		err := s.client.Status().Update(ctx, dogu)
		if err != nil {
			return stepsv3.RequeueWithError(err, "failed to update dogu status")
		}
	}
	return stepsv3.Continue()
}
