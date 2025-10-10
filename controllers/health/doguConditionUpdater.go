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
	name := doguResource.Name
	var err error
	doguResource, err = dcu.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus { //nolint:staticcheck
		meta.SetStatusCondition(&status.Conditions, condition)
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", name, err)
	}

	return nil
}
func (dcu *DoguConditionUpdater) UpdateConditions(ctx context.Context, doguResource *v2.Dogu, conditions []metav1.Condition) error {
	name := doguResource.Name
	var err error
	doguResource, err = dcu.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus { //nolint:staticcheck
		for _, condition := range conditions {
			condition.ObservedGeneration = doguResource.Generation
			meta.SetStatusCondition(&status.Conditions, condition)
		}
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", name, err)
	}
	return nil
}
