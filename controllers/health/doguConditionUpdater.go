package health

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
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
	newDoguResource, err := dcu.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	dcu.setCondition(newDoguResource, condition)
	newDoguResource, err = dcu.doguInterface.UpdateStatus(ctx, newDoguResource, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", doguResource.Name, err)
	}
	doguResource = newDoguResource
	return nil
}
func (dcu *DoguConditionUpdater) UpdateConditions(ctx context.Context, doguResource *v2.Dogu, conditions []metav1.Condition) error {
	newDoguResource, err := dcu.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	for _, condition := range conditions {
		dcu.setCondition(newDoguResource, condition)
	}

	newDoguResource, err = dcu.doguInterface.UpdateStatus(ctx, newDoguResource, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", doguResource.Name, err)
	}
	doguResource = newDoguResource
	return nil
}

func (dcu *DoguConditionUpdater) setCondition(doguResource *v2.Dogu, condition metav1.Condition) {
	for i, existingCondition := range doguResource.Status.Conditions {
		if existingCondition.Type == condition.Type {
			doguResource.Status.Conditions[i] = condition
			return
		}
	}
	doguResource.Status.Conditions = append(doguResource.Status.Conditions, condition)
}
