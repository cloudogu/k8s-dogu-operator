package health

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DoguConditionUpdater struct {
	doguInterface doguInterface
}

func NewDoguConditionUpdater(doguInterface doguClient.DoguInterface) *DoguConditionUpdater {
	return &DoguConditionUpdater{
		doguInterface: doguInterface,
	}
}

func (dcu *DoguConditionUpdater) UpdateCondition(ctx context.Context, doguResource *v2.Dogu, condition metav1.Condition) error {
	condition.ObservedGeneration = doguResource.Generation
	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	name := doguResource.Name
	doguResource, err := dcu.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus {
		return doguResource.Status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", name, err)
	}

	return nil
}
func (dcu *DoguConditionUpdater) UpdateConditions(ctx context.Context, doguResource *v2.Dogu, conditions []metav1.Condition) error {
	for _, condition := range conditions {
		condition.ObservedGeneration = doguResource.Generation
		meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	}
	name := doguResource.Name
	doguResource, err := dcu.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus {
		return doguResource.Status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", name, err)
	}
	return nil
}
