package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonUpgrading = "Upgrading"
)

type PreUpgradeStatusStep struct {
	upgradeChecker upgradeChecker
	doguInterface  doguInterface
}

func (p *PreUpgradeStatusStep) Run(ctx context.Context, resource *v2.Dogu) steps.StepResult {
	isUpgrade, err := p.upgradeChecker.IsUpgrade(ctx, resource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to check if dogu is upgrading: %w", err))
	}

	if isUpgrade {
		resource.Status.Status = v2.DoguStatusUpgrading
		resource.Status.Health = v2.UnavailableHealthStatus

		lastTransitionTime := metav1.Now()
		const message = "The spec version differs from the installed version, therefore an upgrade was scheduled."
		meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
			Type:               v2.ConditionHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonUpgrading,
			Message:            message,
			LastTransitionTime: lastTransitionTime.Rfc3339Copy(),
		})
		meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
			Type:               v2.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonUpgrading,
			Message:            message,
			LastTransitionTime: lastTransitionTime.Rfc3339Copy(),
		})

		resource, err = p.doguInterface.UpdateStatus(ctx, resource, metav1.UpdateOptions{})
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to update dogu status before upgrade: %w", err))
		}
	}

	return steps.Continue()
}

func NewPreUpgradeStatusStep(checker upgrade.Checker, doguInterface doguClient.DoguInterface) *PreUpgradeStatusStep {
	return &PreUpgradeStatusStep{upgradeChecker: checker, doguInterface: doguInterface}
}
