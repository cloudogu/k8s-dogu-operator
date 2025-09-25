package usecase

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguInstallOrChangeUseCase struct {
	client client.Client
	steps  []Step
}

func NewDoguInstallOrChangeUseCase(
	k8sClient client.Client,
	conditionsStep *install.ConditionsStep,
	healthCheckStep *install.HealthCheckStep,
	validationStep *install.ValidationStep,
	pauseReconciliationStep *install.PauseReconciliationStep,
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

	equalDoguDescriptorStep *upgrade.EqualDoguDescriptorsStep,
	upgradeRegisterDoguVersionStep *upgrade.RegisterDoguVersionStep,
	updateDeploymentStep *upgrade.UpdateDeploymentStep,
	deleteExecPodStep *upgrade.DeleteExecPodStep,
	revertStartupProbeStep *upgrade.RevertStartupProbeStep,
	deleteDevelopmentDoguMapStep *upgrade.DeleteDevelopmentDoguMapStep,
	installedVersionStep *upgrade.InstalledVersionStep,
	deploymentUpdaterStep *upgrade.DeploymentUpdaterStep,
	restartDoguStep *upgrade.RestartDoguStep,
	updateStartedAtStep *upgrade.UpdateStartedAtStep,
) *DoguInstallOrChangeUseCase {
	return &DoguInstallOrChangeUseCase{
		client: k8sClient,
		steps: []Step{
			conditionsStep,
			healthCheckStep,
			validationStep,
			pauseReconciliationStep,
			finalizerExistsStep,
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
		err := dicu.client.Get(ctx, types.NamespacedName{Name: doguResource.Name, Namespace: doguResource.Namespace}, doguResource)
		if err != nil {
			return 0, false, err
		}

		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			stepType := getType(s)
			if result.Err != nil {
				logger.Error(result.Err, fmt.Sprintf("reconcile step %s has to requeue: %q", stepType, result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile step %s has to requeue after %d", stepType, result.RequeueAfter))
			}
			return result.RequeueAfter, false, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	logger.Info(fmt.Sprintf("Successfully went through all steps!"))
	return 0, true, nil
}

func getType(val interface{}) string {
	if t := reflect.TypeOf(val); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}
