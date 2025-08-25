package util

import (
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/additionalMount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	remotedogudescriptor "github.com/cloudogu/remote-dogu-descriptor-lib/repository"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudogu/k8s-host-change/pkg/alias"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
)

type ConfigRepositories struct {
	GlobalConfigRepository  *repository.GlobalConfigRepository
	DoguConfigRepository    *repository.DoguConfigRepository
	SensitiveDoguRepository *repository.DoguConfigRepository
}

// ManagerSet contains functors that are repeatedly used by different dogu operator managers.
type ManagerSet struct {
	RestConfig                                 *rest.Config
	CollectApplier                             resource.CollectApplier
	ExecPodFactory                             exec.ExecPodFactory
	FileExtractor                              exec.FileExtractor
	CommandExecutor                            exec.CommandExecutor
	ServiceAccountCreator                      serviceaccount.ServiceAccountCreator
	LocalDoguFetcher                           cesregistry.LocalDoguFetcher
	ResourceDoguFetcher                        cesregistry.ResourceDoguFetcher
	DoguResourceGenerator                      resource.DoguResourceGenerator
	ResourceUpserter                           resource.ResourceUpserter
	DoguRegistrator                            cesregistry.DoguRegistrator
	ImageRegistry                              imageregistry.ImageRegistry
	EcosystemClient                            doguClient.EcoSystemV2Interface
	ClientSet                                  clientSet
	DependencyValidator                        dependencyValidator
	SecurityValidator                          securityValidator
	DoguAdditionalMountValidator               doguAdditionalMountsValidator
	RequirementsGenerator                      requirementsGenerator
	DoguAdditionalMountsInitContainerGenerator additionalMountsInitContainerGenerator
	AdditionalImages                           map[string]string
}

// NewManagerSet creates a new ManagerSet.
func NewManagerSet(restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient doguClient.EcoSystemV2Interface, config *config.OperatorConfig, configRepos ConfigRepositories, applier resource.Applier, additionalImages map[string]string) (*ManagerSet, error) {
	collectApplier := resource.NewCollectApplier(applier)
	commandExecutor := exec.NewCommandExecutor(client, clientSet, clientSet.CoreV1().RESTClient())
	execPodFactory := exec.NewExecPodFactory(client, commandExecutor)
	fileExtractor := exec.NewPodFileExtractor(execPodFactory)
	doguVersionReg := dogu.NewDoguVersionRegistry(clientSet.CoreV1().ConfigMaps(config.Namespace))
	doguDescriptorRepo := dogu.NewLocalDoguDescriptorRepository(clientSet.CoreV1().ConfigMaps(config.Namespace))
	localDoguFetcher := cesregistry.NewLocalDoguFetcher(doguVersionReg, doguDescriptorRepo)
	serviceAccountCreator := serviceaccount.NewCreator(configRepos.SensitiveDoguRepository, localDoguFetcher, commandExecutor, client, clientSet, config.Namespace)
	dependencyValidator := dependency.NewCompositeDependencyValidator(config.Version, localDoguFetcher)
	securityValidator := security.NewValidator()
	doguAdditionalMountsValidator := additionalMount.NewValidator(clientSet.CoreV1().ConfigMaps(config.Namespace), clientSet.CoreV1().Secrets(config.Namespace))

	remoteConfig, err := config.GetRemoteConfiguration()
	if err != nil {
		return nil, err
	}
	doguRemoteRepository, err := remotedogudescriptor.NewRemoteDoguDescriptorRepository(remoteConfig, config.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu repository: %w", err)
	}

	resourceDoguFetcher := cesregistry.NewResourceDoguFetcher(client, doguRemoteRepository)

	requirementsGenerator := resource.NewRequirementsGenerator(configRepos.DoguConfigRepository)
	hostAliasGenerator := alias.NewHostAliasGenerator(configRepos.GlobalConfigRepository)
	securityContextGenerator := resource.NewSecurityContextGenerator()
	doguResourceGenerator := resource.NewResourceGenerator(client.Scheme(), requirementsGenerator, hostAliasGenerator, securityContextGenerator, additionalImages)

	upserter := resource.NewUpserter(client, doguResourceGenerator, config.NetworkPoliciesEnabled)

	doguRegistrator := cesregistry.NewCESDoguRegistrator(doguVersionReg, doguDescriptorRepo)
	imageRegistry := imageregistry.NewCraneContainerImageRegistry()

	return &ManagerSet{
		RestConfig:                   restConfig,
		CollectApplier:               collectApplier,
		ExecPodFactory:               execPodFactory,
		FileExtractor:                fileExtractor,
		CommandExecutor:              commandExecutor,
		ServiceAccountCreator:        serviceAccountCreator,
		LocalDoguFetcher:             localDoguFetcher,
		ResourceDoguFetcher:          resourceDoguFetcher,
		DoguResourceGenerator:        doguResourceGenerator,
		ResourceUpserter:             upserter,
		DoguRegistrator:              doguRegistrator,
		ImageRegistry:                imageRegistry,
		EcosystemClient:              ecosystemClient,
		ClientSet:                    clientSet,
		DependencyValidator:          dependencyValidator,
		SecurityValidator:            securityValidator,
		DoguAdditionalMountValidator: doguAdditionalMountsValidator,
		AdditionalImages:             additionalImages,
		RequirementsGenerator:        requirementsGenerator,
		DoguAdditionalMountsInitContainerGenerator: doguResourceGenerator,
	}, nil
}
