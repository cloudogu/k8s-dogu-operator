package health

import (
	"context"
	"errors"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ReasonStoppingOperator is the reason for conditions set to unknown because the operator is shutting down.
	// v3beta1 does not define this reason, so it is defined locally.
	ReasonStoppingOperator  = "StoppingOperator"
	stoppingOperatorMessage = "The operator is shutting down"
)

// ShutdownHandler is responsible for setting health states to unknown on shutdown of the operator.
type ShutdownHandler struct {
	namespace     string
	doguV3Enabled bool
}

func NewShutdownHandler(config *config.OperatorConfig) *ShutdownHandler {
	return &ShutdownHandler{namespace: config.Namespace, doguV3Enabled: config.DoguV3Enabled}
}

// Handle sets health states of all dogus to unknown.
// v2 dogus get their health and conditions set to unknown, v3 dogus their conditions.
func (s *ShutdownHandler) Handle(ctx context.Context, k8sClient client.Client) error {
	logger := log.FromContext(ctx).WithName("health shutdown handler")
	logger.Info("shutdown detected, handling health status")

	errs := []error{s.handleV2(ctx, k8sClient)}
	if s.doguV3Enabled {
		errs = append(errs, s.handleV3(ctx, k8sClient))
	}
	return errors.Join(errs...)
}

func (s *ShutdownHandler) handleV2(ctx context.Context, k8sClient client.Client) error {
	dogus := &v2.DoguList{}
	if err := k8sClient.List(ctx, dogus, client.InNamespace(s.namespace)); err != nil {
		return fmt.Errorf("failed to list v2 dogus: %w", err)
	}

	var errs []error
	for _, dogu := range dogus.Items {
		// v3 dogus are handled by handleV3
		if !dogu.IsV2() {
			continue
		}

		updateErr := updateStatusWithRetry(ctx, k8sClient, &v2.Dogu{}, client.ObjectKeyFromObject(&dogu), func(current *v2.Dogu) {
			current.Status.Health = v2.UnknownHealthStatus
			for _, conditionType := range []string{
				v2.ConditionReady,
				v2.ConditionHealthy,
				v2.ConditionSupportMode,
				v2.ConditionMeetsMinVolumeSize,
				v2.ConditionPauseReconciliation,
			} {
				meta.SetStatusCondition(&current.Status.Conditions, metav1.Condition{
					Type:               conditionType,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: current.Generation,
					Reason:             ReasonStoppingOperator,
					Message:            stoppingOperatorMessage,
				})
			}
		})

		if updateErr != nil {
			errs = append(errs, fmt.Errorf("failed to set health status and conditions of %q to unknown: %w", dogu.Name, updateErr))
		}
	}
	return errors.Join(errs...)
}

// handleV3 sets all conditions of v3 dogus to unknown.
func (s *ShutdownHandler) handleV3(ctx context.Context, k8sClient client.Client) error {
	logger := log.FromContext(ctx).WithName("health shutdown handler")

	dogus := &v3beta1.DoguList{}
	if err := k8sClient.List(ctx, dogus, client.InNamespace(s.namespace)); err != nil {
		return fmt.Errorf("failed to list v3 dogus: %w", err)
	}

	var errs []error
	for _, dogu := range dogus.Items {
		if dogu.Spec.DoguApiVersion != v3beta1.DoguApiVersionV3 {
			continue
		}

		updateErr := updateStatusWithRetry(ctx, k8sClient, &v3beta1.Dogu{}, client.ObjectKeyFromObject(&dogu), func(current *v3beta1.Dogu) {
			for _, conditionType := range []string{
				v3beta1.ConditionReady,
				v3beta1.ConditionHealthy,
				v3beta1.ConditionPauseReconciliation,
				v3beta1.ConditionStopped,
				v3beta1.ConditionValid,
				v3beta1.ConditionChartAvailable,
				v3beta1.ConditionUpdatePending,
				v3beta1.ConditionSchemaValidationSkipped,
				v3beta1.ConditionExportModeActive,
			} {
				meta.SetStatusCondition(&current.Status.Conditions, metav1.Condition{
					Type:               conditionType,
					Status:             metav1.ConditionUnknown,
					ObservedGeneration: current.Generation,
					Reason:             ReasonStoppingOperator,
					Message:            stoppingOperatorMessage,
				})
			}
		})
		if updateErr != nil {
			err := fmt.Errorf("failed to set conditions of v3 dogu %q to unknown: %w", dogu.Name, updateErr)
			logger.Error(err, "continuing with remaining dogus")
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// updateStatusWithRetry fetches the latest version of the object, modifies it and updates its status,
// retrying on conflict.
func updateStatusWithRetry[T client.Object](ctx context.Context, k8sClient client.Client, obj T, key client.ObjectKey, modify func(T)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k8sClient.Get(ctx, key, obj); err != nil {
			return err
		}
		modify(obj)
		return k8sClient.Status().Update(ctx, obj)
	})
}
