package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguInstallOrChangeUseCase struct {
	steps []step
}

func NewDoguInstallOrChangeUseCase(
	validationStep *install.ValidationStep,
	existsStep *install.FinalizerExistsStep,
	configStep *install.DoguConfigStep,
	doguReferenceStep *install.DoguConfigOwnerReferenceStep,
	sensitiveConfigStep *install.SensitiveConfigStep,
	sensitiveReferenceStep *install.SensitiveConfigOwnerReferenceStep,
	registerDoguVersionStep *install.RegisterDoguVersionStep,
	serviceAccountStep *install.ServiceAccountStep,
	serviceStep *install.ServiceStep,
	customResourceStep *install.CustomK8sResourceStep,
	netPolsStep *install.NetworkPoliciesStep,
	deploymentStep *install.DeploymentStep,
	volumeGeneratorStep *install.VolumeGeneratorStep,
	replicasStep *postinstall.ReplicasStep,
	volumeExpanderStep *postinstall.VolumeExpanderStep,
	ingressAnnotationsStep *postinstall.AdditionalIngressAnnotationsStep,
	securityContextStep *postinstall.SecurityContextStep,
	exportModeStep *postinstall.ExportModeStep,
	supportModeStep *postinstall.SupportModeStep,
	additionalMountsStep *postinstall.AdditionalMountsStep,
	equalDescriptorsStep *upgrade.EqualDoguDescriptorsStep,
	healthStep *upgrade.HealthStep,
) *DoguInstallOrChangeUseCase {
	return &DoguInstallOrChangeUseCase{
		steps: []step{
			validationStep,
			existsStep,
			configStep,
			doguReferenceStep,
			sensitiveConfigStep,
			sensitiveReferenceStep,
			registerDoguVersionStep,
			serviceAccountStep,
			serviceStep,
			customResourceStep,
			netPolsStep,
			deploymentStep,
			volumeGeneratorStep,
			replicasStep,
			volumeExpanderStep,
			ingressAnnotationsStep,
			securityContextStep,
			exportModeStep,
			supportModeStep,
			additionalMountsStep,
			equalDescriptorsStep,
			healthStep,
		},
	}
}

func (dicu *DoguInstallOrChangeUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, error) {
	logger := log.FromContext(ctx).
		WithName("DoguChangeUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range dicu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			logger.Error(result.Err, "reconcile step has to requeue: %w", result.Err)
			return result.RequeueAfter, result.Err
		}
		if !result.Continue {
			break
		}
	}
	return 0, nil
}
