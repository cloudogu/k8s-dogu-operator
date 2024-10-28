package util

import (
	"fmt"
	remotedogudescriptor "github.com/cloudogu/remote-dogu-descriptor-lib/repository"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudogu/k8s-host-change/pkg/alias"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"

	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/serviceaccount"
)

type ConfigRepositories struct {
	GlobalConfigRepository  *repository.GlobalConfigRepository
	DoguConfigRepository    *repository.DoguConfigRepository
	SensitiveDoguRepository *repository.DoguConfigRepository
}

// ManagerSet contains functors that are repeatedly used by different dogu operator managers.
type ManagerSet struct {
	RestConfig            *rest.Config
	CollectApplier        resource.CollectApplier
	FileExtractor         exec.FileExtractor
	CommandExecutor       exec.CommandExecutor
	ServiceAccountCreator serviceaccount.ServiceAccountCreator
	LocalDoguFetcher      cesregistry.LocalDoguFetcher
	ResourceDoguFetcher   cesregistry.ResourceDoguFetcher
	DoguResourceGenerator resource.DoguResourceGenerator
	ResourceUpserter      resource.ResourceUpserter
	DoguRegistrator       cesregistry.DoguRegistrator
	ImageRegistry         imageregistry.ImageRegistry
	EcosystemClient       ecoSystem.EcoSystemV2Interface
	ClientSet             clientSet
	DependencyValidator   dependencyValidator
}

// NewManagerSet creates a new ManagerSet.
func NewManagerSet(restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient ecoSystem.EcoSystemV2Interface, config *config.OperatorConfig, configRepos ConfigRepositories, applier resource.Applier, additionalImages map[string]string) (*ManagerSet, error) {
	collectApplier := resource.NewCollectApplier(applier)
	fileExtractor := exec.NewPodFileExtractor(client, restConfig, clientSet)
	commandExecutor := exec.NewCommandExecutor(client, clientSet, clientSet.CoreV1().RESTClient())
	doguVersionReg := dogu.NewDoguVersionRegistry(clientSet.CoreV1().ConfigMaps(config.Namespace))
	doguDescriptorRepo := dogu.NewLocalDoguDescriptorRepository(clientSet.CoreV1().ConfigMaps(config.Namespace))
	localDoguFetcher := cesregistry.NewLocalDoguFetcher(doguVersionReg, doguDescriptorRepo)
	serviceAccountCreator := serviceaccount.NewCreator(configRepos.SensitiveDoguRepository, localDoguFetcher, commandExecutor, client, clientSet, config.Namespace)
	dependencyValidator := dependency.NewCompositeDependencyValidator(config.Version, localDoguFetcher)

	doguRemoteRepository, err := remotedogudescriptor.NewRemoteDoguDescriptorRepository(config.GetRemoteConfiguration(), config.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu repository: %w", err)
	}

	resourceDoguFetcher := cesregistry.NewResourceDoguFetcher(client, doguRemoteRepository)

	requirementsGenerator := resource.NewRequirementsGenerator(configRepos.DoguConfigRepository)
	hostAliasGenerator := alias.NewHostAliasGenerator(configRepos.GlobalConfigRepository)
	doguResourceGenerator := resource.NewResourceGenerator(client.Scheme(), requirementsGenerator, hostAliasGenerator, additionalImages)

	upserter := resource.NewUpserter(client, doguResourceGenerator)

	doguRegistrator := cesregistry.NewCESDoguRegistrator(doguVersionReg, doguDescriptorRepo)
	imageRegistry := imageregistry.NewCraneContainerImageRegistry(config.DockerRegistry.Username, config.DockerRegistry.Password)

	return &ManagerSet{
		RestConfig:            restConfig,
		CollectApplier:        collectApplier,
		FileExtractor:         fileExtractor,
		CommandExecutor:       commandExecutor,
		ServiceAccountCreator: serviceAccountCreator,
		LocalDoguFetcher:      localDoguFetcher,
		ResourceDoguFetcher:   resourceDoguFetcher,
		DoguResourceGenerator: doguResourceGenerator,
		ResourceUpserter:      upserter,
		DoguRegistrator:       doguRegistrator,
		ImageRegistry:         imageRegistry,
		EcosystemClient:       ecosystemClient,
		ClientSet:             clientSet,
		DependencyValidator:   dependencyValidator,
	}, nil
}
