package usecase

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguInstallOrChangeUseCase struct {
	steps []Step
}

func NewDoguInstallOrChangeUseCase(
	conditionsStep *install.ConditionsStep,
	healthCheckStep *install.HealthCheckStep,
	validationStep *install.ValidationStep,
	pauseReconcilationStep *install.PauseReconcilationStep,
	finalizerExistsStep *install.FinalizerExistsStep,
	createDoguConfigStep install.CreateDoguConfigStep,
	doguConfigOwnerReferenceStep install.DoguConfigOwnerReferenceStep,
	createSensitiveDoguConfigStep install.CreateSensitiveDoguConfigStep,
	sensitiveDoguConfigOwnerReferenceStep install.SensitiveDoguConfigOwnerReferenceStep,
	registerDoguVersionStep *install.RegisterDoguVersionStep,
	localDoguDescriptorOwnerReferenceStep install.LocalDoguDescriptorOwnerReferenceStep,
	serviceAccountStep *install.ServiceAccountStep,
	serviceStep *install.ServiceStep,
	execPodCreateStep *install.ExecPodCreateStep,
	customK8sResourceStep *install.CustomK8sResourceStep,
	volumeGeneratorStep *install.VolumeGeneratorStep,
	networkPoliciesStep *install.NetworkPoliciesStep,
	deploymentStep *install.DeploymentStep,

	replicasStep *postinstall.ReplicasStep,
	volumeExpanderStep *postinstall.VolumeExpanderStep,
	additionalIngressAnnotationsStep *postinstall.AdditionalIngressAnnotationsStep,
	securityContextStep *postinstall.SecurityContextStep,
	exportModeStep *postinstall.ExportModeStep,
	supportModeStep *postinstall.SupportModeStep,
	additionalMountsStep *postinstall.AdditionalMountsStep,
	restartDoguStep *postinstall.RestartDoguStep,

	equalDoguDescriptorStep *upgrade.EqualDoguDescriptorsStep,
	upgradeRegisterDoguVersionStep *upgrade.RegisterDoguVersionStep,
	updateDeploymentStep *upgrade.UpdateDeploymentStep,
	deleteExecPodStep *upgrade.DeleteExecPodStep,
	revertStartupProbeStep *upgrade.RevertStartupProbeStep,
	deleteDevelopmentDoguMapStep *upgrade.DeleteDevelopmentDoguMapStep,
	installedVersionStep *upgrade.InstalledVersionStep,
	deploymentUpdaterStep *upgrade.DeploymentUpdaterStep,
	updateStartedAtStep *upgrade.UpdateStartedAtStep,
) *DoguInstallOrChangeUseCase {
	return &DoguInstallOrChangeUseCase{
		steps: []Step{
			conditionsStep,
			healthCheckStep,
			validationStep,
			finalizerExistsStep,
			pauseReconcilationStep,
			createDoguConfigStep,
			doguConfigOwnerReferenceStep,
			createSensitiveDoguConfigStep,
			sensitiveDoguConfigOwnerReferenceStep,
			registerDoguVersionStep,
			localDoguDescriptorOwnerReferenceStep,
			serviceAccountStep,
			serviceStep,
			execPodCreateStep,
			customK8sResourceStep,
			volumeGeneratorStep,
			networkPoliciesStep,

			deploymentStep,
			replicasStep,
			volumeExpanderStep,
			additionalIngressAnnotationsStep,
			securityContextStep,
			exportModeStep,
			supportModeStep,
			additionalMountsStep,
			restartDoguStep,

			equalDoguDescriptorStep,
			upgradeRegisterDoguVersionStep,
			updateDeploymentStep,
			deleteExecPodStep,
			revertStartupProbeStep,
			deleteDevelopmentDoguMapStep,
			installedVersionStep,
			deploymentUpdaterStep,
			updateStartedAtStep,
		},
	}
}

func (dicu *DoguInstallOrChangeUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error) {
	logger := log.FromContext(ctx).
		WithName("DoguChangeUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range dicu.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			if result.Err != nil {
				logger.Error(result.Err, fmt.Sprintf("reconcile Step has to requeue: %q", result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile Step has to requeue after %d", result.RequeueAfter))
			}
			return result.RequeueAfter, true, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	logger.Info(fmt.Sprintf("Successfully went through all steps!"))
	return 0, true, nil
}
