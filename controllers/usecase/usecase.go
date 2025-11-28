package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
)

type DoguUseCase struct {
	steps []Step
}

func NewDoguDeleteUseCase(
	statusStep *deletion.StatusStep,
	serviceAccountRemoverStep *deletion.ServiceAccountRemoverStep,
	deleteOutOfHealthConfigMapStep *deletion.DeleteOutOfHealthConfigMapStep,
	removeSensitiveDoguConfigStep deletion.RemoveSensitiveDoguConfigStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
) *DoguUseCase {
	return &DoguUseCase{
		steps: []Step{
			statusStep,
			serviceAccountRemoverStep,
			deleteOutOfHealthConfigMapStep,
			removeSensitiveDoguConfigStep,
			removeFinalizerStep,
		}}
}

//nolint:funlen
func NewDoguInstallOrChangeUseCase(
	conditionsStep *install.InitializeConditionsStep,
	healthCheckStep *install.HealthCheckStep,
	fetchRemoteDoguDescriptorStep *install.FetchRemoteDoguDescriptorStep,
	validationStep *install.ValidationStep,
	pauseReconciliationStep *install.PauseReconciliationStep,
	finalizerExistsStep *install.CreateFinalizerStep,
	createDoguConfigStep install.CreateDoguConfigStep,
	doguConfigOwnerReferenceStep install.DoguConfigOwnerReferenceStep,
	createSensitiveDoguConfigStep install.CreateSensitiveDoguConfigStep,
	sensitiveDoguConfigOwnerReferenceStep install.SensitiveDoguConfigOwnerReferenceStep,
	registerDoguVersionStep *install.RegisterDoguVersionStep,
	localDoguDescriptorOwnerReferenceStep install.LocalDoguDescriptorOwnerReferenceStep,
	removeServiceAccountStep *install.RemoveServiceAccountStep,
	serviceAccountStep *install.ServiceAccountStep,
	serviceStep *install.ServiceStep,
	execPodCreateStep *install.CreateExecPodStep,
	customK8sResourceStep *install.CustomK8sResourceStep,
	volumeGeneratorStep *install.CreateVolumeStep,
	networkPoliciesStep *install.NetworkPoliciesStep,
	deploymentStep *install.CreateDeploymentStep,

	replicasStep *postinstall.StartStopStep,
	volumeExpanderStep *postinstall.VolumeExpanderStep,
	additionalIngressAnnotationsStep *postinstall.AdditionalIngressAnnotationsStep,
	securityContextStep *postinstall.SecurityContextStep,
	exportModeStep *postinstall.ExportModeStep,
	supportModeStep *postinstall.SupportModeStep,
	additionalMountsStep *postinstall.AdditionalMountsStep,

	preUpgradeStatusStep *upgrade.PreUpgradeStatusStep,
	updateDeploymentStep *upgrade.UpdateDeploymentVersionStep,
	deleteExecPodStep *upgrade.DeleteExecPodStep,
	revertStartupProbeStep *upgrade.PostUpgradeStep,
	deploymentUpdaterStep *upgrade.RegenerateDeploymentStep,
	upgradeRegisterDoguVersionStep *upgrade.RegisterDoguVersionStep,
	installedVersionStep *upgrade.InstalledVersionStep,
	updateStartedAtStep *upgrade.UpdateStartedAtStep,
	restartDoguStep *upgrade.RestartAfterConfigChangeStep,
	retroactiveServiceAccountStep *upgrade.RetroactiveServiceAccountStep,
) *DoguUseCase {
	return &DoguUseCase{
		steps: []Step{
			conditionsStep,
			healthCheckStep,
			fetchRemoteDoguDescriptorStep,
			validationStep,
			pauseReconciliationStep,
			finalizerExistsStep,
			createDoguConfigStep,
			doguConfigOwnerReferenceStep,
			createSensitiveDoguConfigStep,
			sensitiveDoguConfigOwnerReferenceStep,
			registerDoguVersionStep,
			localDoguDescriptorOwnerReferenceStep,
			removeServiceAccountStep,
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

			preUpgradeStatusStep,
			updateDeploymentStep,
			deleteExecPodStep,
			revertStartupProbeStep,
			installedVersionStep,
			deploymentUpdaterStep,
			upgradeRegisterDoguVersionStep,
			updateStartedAtStep,
			restartDoguStep,
			retroactiveServiceAccountStep,
		},
	}
}

func (duc *DoguUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, bool, error) {
	for _, s := range duc.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			return result.RequeueAfter, false, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	return 0, true, nil
}
