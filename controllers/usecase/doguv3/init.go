package doguv3

import (
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/migration"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/charts"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/registry"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDoguV3UseCases assembles the DoguV3 install-or-change and delete use-cases from the shared
// infrastructure fx provides. It performs the cluster capability discovery and dogu-registry
// client construction eagerly, so a failure here aborts operator startup (fail-fast).
func NewDoguV3UseCases(
	k8sClient client.Client,
	apiReader client.Reader, // for uncached reading
	recorder record.EventRecorder,
	restConfig *rest.Config,
	clientSet kubernetes.Interface,
	operatorConfig *config.OperatorConfig,
) (controllers.DoguV3InstallOrChangeUseCase, controllers.DoguV3DeleteUseCase, error) {
	registryReader, err := registry.NewDoguRegistryReader(operatorConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build v3 dogu registry reader: %w", err)
	}

	capabilities, err := charts.NewCapabilities(clientSet)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover chart capabilities: %w", err)
	}

	releaseReader := flux.NewHelmReleaseReader(k8sClient, clientSet)
	chartService := charts.NewService(charts.NewChartProvider(k8sClient, charts.NewHTTPClient()), restConfig, capabilities, releaseReader)
	assembler := values.NewAssembler(k8sClient)
	resizeChecker := pvc.NewResizeChecker(k8sClient)

	finalizerStep := install.NewCreateFinalizerStep(k8sClient)
	ociStep := install.NewEnsureOCIRepositoryStep(k8sClient, registryReader, recorder)
	waitStep := install.NewWaitForOCIRepositoryReadyStep(k8sClient, recorder)
	validateStep := install.NewValidateChartStep(chartService, assembler, resizeChecker, k8sClient)
	resizePlanner := install.NewVolumeResizePlanner(chartService, assembler, resizeChecker, apiReader)
	resizePVCStep := install.NewResizePVCStep(resizePlanner, apiReader, k8sClient, recorder, operatorConfig.OfflineVolumeExpansion)
	suspendStep := install.NewSuspendHelmReleaseStep(k8sClient, resizePlanner, recorder)
	helmReleaseStep := install.NewEnsureHelmReleaseStep(k8sClient, operatorConfig, chartService, assembler, recorder)
	helmReleaseStatusStep := install.NewHelmReleaseStatusStep(k8sClient, chartService, assembler, operatorConfig, releaseReader, recorder)
	exportModeStep := install.NewExportModeStep(k8sClient, recorder)
	updatePendingStep := install.NewUpdatePendingStep(k8sClient)
	volumeStatusStep := install.NewVolumeStatusStep(chartService, assembler, workloads.NewDiscovery(k8sClient), k8sClient, recorder)

	unsuspendStep := deletion.NewUnsuspendHelmReleaseStep(k8sClient, recorder)
	deleteHelmReleaseStep := deletion.NewDeleteHelmReleaseStep(k8sClient, recorder)
	deleteOCIRepositoryStep := deletion.NewDeleteOCIRepositoryStep(k8sClient, recorder)
	removeFinalizerStep := deletion.NewRemoveFinalizerStep(k8sClient)

	detector := dogumigration.NewDetector(k8sClient)
	migrationPreconditionsStep := migration.NewMigrationPreconditionsStep(detector, recorder)
	migrationPendingStep := migration.NewMigrationPendingStep(detector, recorder)

	installUseCase := NewDoguInstallOrChangeUseCase(finalizerStep, ociStep, waitStep, validateStep, suspendStep, resizePVCStep, helmReleaseStep, helmReleaseStatusStep, updatePendingStep, exportModeStep, volumeStatusStep, k8sClient, recorder)
	migrationUseCase := NewDoguMigrationUseCase(finalizerStep, migrationPreconditionsStep, ociStep, waitStep, validateStep, migrationPendingStep, k8sClient, recorder)
	deleteUseCase := NewDoguDeleteUseCase(unsuspendStep, deleteHelmReleaseStep, deleteOCIRepositoryStep, removeFinalizerStep, k8sClient, recorder)

	changeHandler := NewDoguChangeHandler(detector, installUseCase, migrationUseCase)

	return changeHandler, deleteUseCase, nil
}
