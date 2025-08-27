package usecase

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
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

func NewDoguInstallOrChangeUseCase(client client.Client, mgrSet *util.ManagerSet, configRepos util.ConfigRepositories, eventRecorder record.EventRecorder, namespace string) *DoguInstallOrChangeUseCase {
	return &DoguInstallOrChangeUseCase{
		steps: []step{
			install.NewValidationStep(mgrSet),
			install.NewFinalizerExistsStep(),
			install.NewDoguConfigStep(configRepos),
			install.NewDoguConfigOwnerReferenceStep(configRepos),
			install.NewSensitiveConfigStep(configRepos),
			install.NewSensitiveConfigOwnerReferenceStep(configRepos),
			install.NewRegisterDoguVersionStep(mgrSet),
			install.NewServiceAccountStep(mgrSet),
			install.NewServiceStep(mgrSet, namespace),
			install.NewExecPodCreateStep(client, mgrSet, eventRecorder),
			install.NewCustomK8sResourceStep(mgrSet, eventRecorder),
			install.NewNetworkPoliciesStep(mgrSet),
			install.NewDeploymentStep(client, mgrSet),
			install.NewVolumeGeneratorStep(mgrSet),
			postinstall.NewReplicasStep(client, mgrSet, namespace),
			postinstall.NewVolumeExpanderStep(client, mgrSet, namespace),
			postinstall.NewAdditionalIngressAnnotationsStep(client),
			postinstall.NewSecurityContextStep(mgrSet, namespace),
			postinstall.NewExportModeStep(mgrSet, namespace, eventRecorder),
			postinstall.NewSupportModeStep(client, mgrSet, eventRecorder),
			postinstall.NewAdditionalMountsStep(mgrSet, namespace),
			upgrade.NewEqualDoguDescriptorsStep(mgrSet),
			upgrade.NewHealthStep(mgrSet),
			upgrade.NewRegisterDoguVersionStep(mgrSet),
			upgrade.NewUpdateDeploymentStep(client, mgrSet, namespace),
			upgrade.NewDeleteExecPodStep(mgrSet),
			upgrade.NewRevertStartupProbeStep(client, mgrSet, namespace),
			upgrade.NewDeleteDevelopmentDoguMapStep(client, mgrSet),
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
