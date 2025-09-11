package usecase

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguInstallOrChangeUseCase struct {
	steps []step
}

func NewDoguInstallOrChangeUseCase(client client.Client, mgrSet *util.ManagerSet, configRepos util.ConfigRepositories, eventRecorder record.EventRecorder, namespace string, doguHealthStatusUpdater health.DoguHealthStatusUpdater, doguRestartManager doguRestartManager, availabilityChecker *health.AvailabilityChecker) *DoguInstallOrChangeUseCase {
	return &DoguInstallOrChangeUseCase{
		steps: []step{
			install.NewConditionsStep(mgrSet, namespace),
			install.NewHealthCheckStep(client, availabilityChecker, doguHealthStatusUpdater, mgrSet, namespace),
			install.NewValidationStep(mgrSet),
			install.NewFinalizerExistsStep(),
			// Dogu config steps
			install.NewCreateConfigStep(configRepos.DoguConfigRepository),
			install.NewOwnerReferenceStep(configRepos.DoguConfigRepository),
			// Sensitive dogu config steps
			install.NewCreateConfigStep(configRepos.SensitiveDoguRepository),
			install.NewOwnerReferenceStep(configRepos.SensitiveDoguRepository),
			install.NewRegisterDoguVersionStep(mgrSet),
			// Set owner reference for local dogu descriptor
			install.NewOwnerReferenceStep(mgrSet.LocalDoguDescriptorRepository),
			install.NewServiceAccountStep(mgrSet),
			install.NewServiceStep(mgrSet, namespace),
			install.NewExecPodCreateStep(client, mgrSet, eventRecorder),
			install.NewCustomK8sResourceStep(mgrSet, eventRecorder),
			install.NewVolumeGeneratorStep(mgrSet, namespace),
			install.NewNetworkPoliciesStep(mgrSet),
			install.NewDeploymentStep(client, mgrSet),
			postinstall.NewReplicasStep(client, mgrSet, namespace),
			postinstall.NewVolumeExpanderStep(client, mgrSet, namespace),
			postinstall.NewAdditionalIngressAnnotationsStep(client),
			postinstall.NewSecurityContextStep(mgrSet, namespace),
			postinstall.NewExportModeStep(mgrSet, namespace, eventRecorder),
			postinstall.NewSupportModeStep(client, mgrSet, eventRecorder, namespace),
			postinstall.NewAdditionalMountsStep(mgrSet, namespace),
			postinstall.NewRestartDoguStep(client, mgrSet, namespace, configRepos, doguRestartManager),
			upgrade.NewEqualDoguDescriptorsStep(mgrSet),
			upgrade.NewRegisterDoguVersionStep(mgrSet),
			upgrade.NewUpdateDeploymentStep(client, mgrSet, namespace),
			upgrade.NewDeleteExecPodStep(mgrSet),
			upgrade.NewRevertStartupProbeStep(client, mgrSet, namespace),
			upgrade.NewDeleteDevelopmentDoguMapStep(client, mgrSet),
			upgrade.NewInstalledVersionStep(mgrSet, namespace),
			upgrade.NewDeploymentUpdaterStep(client, mgrSet, namespace),
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
				logger.Error(result.Err, fmt.Sprintf("reconcile step has to requeue: %q", result.Err))
			} else {
				logger.Info(fmt.Sprintf("reconcile step has to requeue after %d", result.RequeueAfter))
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
