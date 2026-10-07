package doguv3

import (
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/charts"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
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

	chartService := charts.NewService(charts.NewChartProvider(k8sClient, charts.NewHTTPClient()), restConfig, capabilities, flux.NewHelmReleaseReader(k8sClient))
	assembler := values.NewAssembler(k8sClient)

	ociStep := install.NewEnsureOCIRepositoryStep(k8sClient, registryReader, recorder)
	waitStep := install.NewWaitForOCIRepositoryReadyStep(k8sClient, recorder)
	validateStep := install.NewValidateChartStep(chartService, assembler, k8sClient)
	helmReleaseStep := install.NewEnsureHelmReleaseStep(k8sClient, operatorConfig, chartService, recorder)

	installUseCase := NewDoguInstallOrChangeUseCase(ociStep, waitStep, validateStep, helmReleaseStep, k8sClient, recorder)
	deleteUseCase := NewDoguDeleteUseCase(k8sClient)

	return installUseCase, deleteUseCase, nil
}
