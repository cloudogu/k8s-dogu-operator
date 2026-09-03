package health

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ShutdownHandler is responsible for setting health states to unknown on shutdown of the operator.
type ShutdownHandler struct {
	doguInterface doguv2.DoguInterface
}

func NewShutdownHandler(doguInterface doguv2.DoguInterface) *ShutdownHandler {
	return &ShutdownHandler{doguInterface: doguInterface}
}

// Handle waits for the context to be cancelled and then sets health states to unknown.
func (s *ShutdownHandler) Handle(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("health shutdown handler")
	logger.Info("shutdown detected, handling health status")

	dogus, err := s.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var errs []error
	for _, dogu := range dogus.Items {
		// Skip Non-V2-Dogus for now
		if !dogu.IsV2() {
			continue
		}

		_, updateErr := s.doguInterface.UpdateStatusWithRetry(ctx, &dogu, func(status v2.DoguStatus) v2.DoguStatus {
			status.Health = v2.UnknownHealthStatus
			reason := "StoppingOperator"
			message := "The operator is shutting down"
			conditions := []metav1.Condition{
				{
					Type:               v2.ConditionReady,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: dogu.Generation,
					Reason:             reason,
					Message:            message,
				},
				{
					Type:               v2.ConditionHealthy,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: dogu.Generation,
					Reason:             reason,
					Message:            message,
				},
				{
					Type:               v2.ConditionSupportMode,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: dogu.Generation,
					Reason:             reason,
					Message:            message,
				},
				{
					Type:               v2.ConditionMeetsMinVolumeSize,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: dogu.Generation,
					Reason:             reason,
					Message:            message,
				},
			}
			for _, condition := range conditions {
				meta.SetStatusCondition(&status.Conditions, condition)
			}

			return status
		}, metav1.UpdateOptions{})
		if updateErr != nil {
			errs = append(errs, fmt.Errorf("failed to set health status and conditions of %q to unknown: %w", dogu.Name, updateErr))
		}
	}
	return errors.Join(errs...)
}
