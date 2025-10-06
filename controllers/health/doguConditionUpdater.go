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
	newDoguResource, err := dcu.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	meta.SetStatusCondition(&newDoguResource.Status.Conditions, condition)
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
		meta.SetStatusCondition(&newDoguResource.Status.Conditions, condition)
	}

	newDoguResource, err = dcu.doguInterface.UpdateStatus(ctx, newDoguResource, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s: %w", doguResource.Name, err)
	}
	doguResource = newDoguResource
	return nil
}
