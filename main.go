package main

import (
	"go.uber.org/fx"

	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlMan "sigs.k8s.io/controller-runtime/pkg/manager"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/cloudogu/ces-commons-lib/dogu"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/additionalMount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/garbagecollection"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/logging"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	upgradeSteps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
	"github.com/cloudogu/k8s-registry-lib/repository"
)

var (
	// Version of the application
	Version = "0.0.0"
)

// newVersion is a constructor to inject the version without import cycles
func newVersion() config.Version {
	return config.Version(Version)
}

func main() {
	fx.New(options()...).Run()
}

//nolint:funlen
func options() []fx.Option {
	return []fx.Option{
		fx.Provide(
			newVersion,
			logging.NewLogger,
			initfx.NewOperatorConfig,
			initfx.GetArgs,

			// k8s dependencies
			initfx.NewManagerOptions,
			ctrl.GetConfig,
			initfx.NewScheme,
			fx.Annotate(initfx.NewK8sClient, fx.As(new(client.Client))),
			fx.Annotate(initfx.NewKubernetesClientSet, fx.As(new(kubernetes.Interface))),
			fx.Annotate(initfx.NewRestClient, fx.As(new(rest.Interface))),
			fx.Annotate(initfx.NewConfigMapInterface, fx.As(new(v1.ConfigMapInterface)), fx.As(new(repository.ConfigMapClient))),
			fx.Annotate(initfx.NewSecretInterface, fx.As(new(v1.SecretInterface)), fx.As(new(repository.SecretClient))),
			fx.Annotate(initfx.NewDeploymentInterface, fx.As(new(appsv1.DeploymentInterface))),
			fx.Annotate(initfx.NewPodInterface, fx.As(new(v1.PodInterface))),
			fx.Annotate(initfx.NewServiceInterface, fx.As(new(v1.ServiceInterface))),
			fx.Annotate(initfx.NewPersistentVolumeClaimInterface, fx.As(new(v1.PersistentVolumeClaimInterface))),
			fx.Annotate(initfx.NewEcoSystemClientSet, fx.As(new(doguClient.EcoSystemV2Interface))),
			fx.Annotate(initfx.NewDoguInterface, fx.As(new(doguClient.DoguInterface))),
			fx.Annotate(initfx.NewDoguRestartInterface, fx.As(new(doguClient.DoguRestartInterface))),
			fx.Annotate(health.NewShutdownHandler, fx.As(new(health.HealthShutdownHandler))),

			fx.Annotate(initfx.NewControllerManager, fx.As(new(ctrlMan.Manager))),
			fx.Annotate(initfx.NewEventRecorder, fx.As(new(record.EventRecorder))),
			fx.Annotate(controllers.NewDoguRequeueHandler, fx.As(new(controllers.RequeueHandler))),

			// our own dependencies
			fx.Annotate(health.NewAvailabilityChecker, fx.As(new(health.DeploymentAvailabilityChecker))),
			fx.Annotate(health.NewDoguStatusUpdater, fx.As(new(health.DoguHealthStatusUpdater))),
			fx.Annotate(initfx.NewCollectApplier, fx.As(new(initfx.CollectApplier)), fx.As(new(resource.CollectApplier))),
			initfx.GetAdditionalImages,
			fx.Annotate(initfx.NewCommandExecutor, fx.As(new(exec.CommandExecutor))),
			fx.Annotate(exec.NewExecPodFactory, fx.As(new(exec.ExecPodFactory))),
			fx.Annotate(exec.NewPodFileExtractor, fx.As(new(exec.FileExtractor))),
			fx.Annotate(initfx.NewDoguVersionRegistry, fx.As(new(dogu.VersionRegistry))),
			// provide twice, tagged as well as untagged
			fx.Annotate(
				initfx.NewLocalDoguDescriptorRepository,
				fx.As(new(dogu.LocalDoguDescriptorRepository)),
				fx.As(new(initfx.LocalDoguDescriptorRepository)),
			),
			fx.Annotate(
				initfx.NewLocalDoguDescriptorRepository,
				fx.As(new(initfx.OwnerReferenceSetter)),
				fx.ResultTags(`name:"localDoguDescriptorRepository"`),
			),
			fx.Annotate(initfx.NewLocalDoguFetcher, fx.As(new(cesregistry.LocalDoguFetcher))),
			fx.Annotate(repository.NewGlobalConfigRepository, fx.As(new(resource.GlobalConfigRepository))),
			// provide twice, tagged as well as untagged
			fx.Annotate(
				repository.NewDoguConfigRepository,
				fx.As(new(resource.DoguConfigRepository)),
			),
			fx.Annotate(
				repository.NewDoguConfigRepository,
				fx.As(new(initfx.DoguConfigRepository)),
				fx.As(new(initfx.OwnerReferenceSetter)),
				fx.ResultTags(`name:"normalDoguConfig"`),
			),
			// provide twice, tagged as well as untagged
			fx.Annotate(
				repository.NewSensitiveDoguConfigRepository,
				fx.As(new(serviceaccount.SensitiveDoguConfigRepository)),
			),
			fx.Annotate(
				repository.NewSensitiveDoguConfigRepository, fx.As(new(initfx.DoguConfigRepository)),
				fx.As(new(initfx.OwnerReferenceSetter)),
				fx.ResultTags(`name:"sensitiveDoguConfig"`),
			),
			fx.Annotate(serviceaccount.NewCreator, fx.As(new(serviceaccount.ServiceAccountCreator))),
			fx.Annotate(serviceaccount.NewRemover, fx.As(new(serviceaccount.ServiceAccountRemover))),
			fx.Annotate(dependency.NewCompositeDependencyValidator, fx.As(new(dependency.Validator))),
			fx.Annotate(security.NewValidator, fx.As(new(security.Validator))),
			fx.Annotate(additionalMount.NewValidator, fx.As(new(additionalMount.Validator))),
			fx.Annotate(initfx.NewRemoteDoguDescriptorRepository, fx.As(new(dogu.RemoteDoguDescriptorRepository))),
			fx.Annotate(initfx.NewResourceDoguFetcher, fx.As(new(cesregistry.ResourceDoguFetcher))),
			fx.Annotate(resource.NewRequirementsGenerator, fx.As(new(resource.RequirementsGenerator))),
			fx.Annotate(initfx.NewHostAliasGenerator, fx.As(new(resource.HostAliasGenerator))),
			fx.Annotate(resource.NewSecurityContextGenerator, fx.As(new(resource.SecurityContextGenerator))),
			fx.Annotate(resource.NewResourceGenerator, fx.As(new(resource.DoguResourceGenerator))),
			fx.Annotate(resource.NewUpserter, fx.As(new(resource.ResourceUpserter)), fx.As(new(upgradeSteps.ResourceUpserter))),
			fx.Annotate(cesregistry.NewCESDoguRegistrator, fx.As(new(cesregistry.DoguRegistrator))),
			fx.Annotate(initfx.NewImageRegistry, fx.As(new(imageregistry.ImageRegistry))),
			fx.Annotate(manager.NewDoguRestartManager, fx.As(new(manager.DoguRestartManager))),
			fx.Annotate(garbagecollection.NewDoguRestartGarbageCollector, fx.As(new(controllers.DoguRestartGarbageCollector))),
			fx.Annotate(health.NewDoguConditionUpdater, fx.As(new(install.ConditionUpdater))),
			fx.Annotate(health.NewDoguChecker, fx.As(new(health.DoguHealthChecker))),
			fx.Annotate(manager.NewDoguExportManager, fx.As(new(manager.DoguExportManager))),
			fx.Annotate(manager.NewDoguSupportManager, fx.As(new(manager.SupportManager))),
			fx.Annotate(manager.NewDoguAdditionalMountManager, fx.As(new(manager.AdditionalMountManager))),
			fx.Annotate(manager.NewDeploymentManager, fx.As(new(manager.DeploymentManager))),
			fx.Annotate(upgrade.NewChecker, fx.As(new(upgrade.Checker))),
			controllers.NewDoguEvents,
			controllers.NewDoguEventsIn,
			controllers.NewDoguEventsOut,

			// delete steps
			deletion.NewStatusStep,
			deletion.NewServiceAccountRemoverStep,
			deletion.NewDeleteOutOfHealthConfigMapStep,
			fx.Annotate(deletion.NewRemoveDoguConfigStep, fx.ParamTags(`name:"sensitiveDoguConfig"`), fx.As(new(deletion.RemoveSensitiveDoguConfigStep))),
			deletion.NewRemoveFinalizerStep,

			// install or change steps
			install.NewInitializeConditionsStep,
			install.NewHealthCheckStep,
			install.NewFetchRemoteDoguDescriptorStep,
			install.NewValidationStep,
			install.NewPauseReconciliationStep,
			install.NewCreateFinalizerStep,
			// Dogu config steps
			fx.Annotate(
				install.NewCreateConfigStep,
				fx.ParamTags(`name:"normalDoguConfig"`),
				fx.As(new(install.CreateDoguConfigStep)),
			),
			fx.Annotate(
				install.NewOwnerReferenceStep,
				fx.ParamTags(`name:"normalDoguConfig"`),
				fx.As(new(install.DoguConfigOwnerReferenceStep)),
			),
			// Sensitive dogu config steps
			fx.Annotate(
				install.NewCreateConfigStep,
				fx.ParamTags(`name:"sensitiveDoguConfig"`),
				fx.As(new(install.CreateSensitiveDoguConfigStep)),
			),
			fx.Annotate(
				install.NewOwnerReferenceStep,
				fx.ParamTags(`name:"sensitiveDoguConfig"`),
				fx.As(new(install.SensitiveDoguConfigOwnerReferenceStep)),
			),
			// Create local dogu descriptor and set owner reference
			install.NewRegisterDoguVersionStep,
			fx.Annotate(
				install.NewOwnerReferenceStep,
				fx.ParamTags(`name:"localDoguDescriptorRepository"`),
				fx.As(new(install.LocalDoguDescriptorOwnerReferenceStep)),
			),
			install.NewRemoveServiceAccountStep,
			install.NewServiceAccountStep,
			install.NewServiceStep,
			install.NewCreateExecPodStep,
			install.NewCustomK8sResourceStep,
			install.NewCreateVolumeStep,
			install.NewNetworkPoliciesStep,
			install.NewCreateDeploymentStep,
			postinstall.NewStartStopStep,
			postinstall.NewVolumeExpanderStep,
			postinstall.NewMismatchedStorageClassWarningStep,
			postinstall.NewAdditionalIngressAnnotationsStep,
			postinstall.NewSecurityContextStep,
			postinstall.NewExportModeStep,
			postinstall.NewSupportModeStep,
			postinstall.NewAdditionalMountsStep,
			fx.Annotate(upgradeSteps.NewRestartAfterConfigChangeStep, fx.ParamTags(`name:"normalDoguConfig"`, `name:"sensitiveDoguConfig"`, "", "", "")),
			upgradeSteps.NewPreUpgradeStatusStep,
			upgradeSteps.NewRegisterDoguVersionStep,
			upgradeSteps.NewUpdateDeploymentVersionStep,
			upgradeSteps.NewDeleteExecPodStep,
			upgradeSteps.NewPostUpgradeStep,
			upgradeSteps.NewInstalledVersionStep,
			upgradeSteps.NewRegenerateDeploymentStep,
			upgradeSteps.NewUpdateStartedAtStep,
			upgradeSteps.NewRetroactiveServiceAccountStep,

			// use-cases
			fx.Annotate(
				usecase.NewDoguDeleteUseCase,
				fx.As(new(controllers.DoguDeleteUseCase)),
				fx.ResultTags(`name:"doguDeleteUseCase"`),
			),
			fx.Annotate(
				usecase.NewDoguDeleteUseCase,
				fx.As(new(controllers.DoguDeleteUseCase)),
			),
			fx.Annotate(
				usecase.NewDoguInstallOrChangeUseCase,
				fx.As(new(controllers.DoguInstallOrChangeUseCase)),
				fx.ResultTags(`name:"doguInstallOrChangeUseCase"`),
			),
			fx.Annotate(
				usecase.NewDoguInstallOrChangeUseCase,
				fx.As(new(controllers.DoguInstallOrChangeUseCase)),
			),

			// reconcilers
			fx.Annotate(controllers.NewDoguReconciler, fx.ParamTags("", `name:"doguInstallOrChangeUseCase"`, `name:"doguDeleteUseCase"`, "", "", "", "", "")),
			controllers.NewGlobalConfigReconciler,
			controllers.NewDoguRestartReconciler,

			// runners
			health.NewStartupHandler,
			health.NewShutdownHandler,
		),
		// the empty invoke functions tell fx to instantiate these structs even if nothing depends on them.
		// reconcilers and runners are the last in the dependency chain so we have to invoke them here.
		fx.Invoke(
			func(*controllers.DoguReconciler) {
				// creates a fx dependency on the DoguReconciler
			},
			func(*controllers.DoguRestartReconciler) {
				// creates a fx dependency on the DoguRestartReconciler
			},
			func(*controllers.GlobalConfigReconciler) {
				// creates a fx dependency on the GlobalConfigReconciler
			},

			func(*health.StartupHandler) {
				// creates a fx dependency on the StartupHandler
			},
		),
	}
}
