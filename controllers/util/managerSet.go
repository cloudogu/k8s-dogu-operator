package util

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-host-change/pkg/alias"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
)

type ConfigRepositories struct {
	GlobalConfigRepository  *repository.GlobalConfigRepository
	DoguConfigRepository    *repository.DoguConfigRepository
	SensitiveDoguRepository *repository.DoguConfigRepository
}

// ManagerSet contains functors that are repeatedly used by different dogu operator managers.
type ManagerSet struct {
	RestConfig            *rest.Config
	CollectApplier        cloudogu.CollectApplier
	FileExtractor         cloudogu.FileExtractor
	CommandExecutor       cloudogu.CommandExecutor
	ServiceAccountCreator cloudogu.ServiceAccountCreator
	LocalDoguFetcher      cloudogu.LocalDoguFetcher
	ResourceDoguFetcher   cloudogu.ResourceDoguFetcher
	DoguResourceGenerator cloudogu.DoguResourceGenerator
	ResourceUpserter      cloudogu.ResourceUpserter
	DoguRegistrator       cloudogu.DoguRegistrator
	ImageRegistry         cloudogu.ImageRegistry
	EcosystemClient       cloudogu.EcosystemInterface
	ClientSet             thirdParty.ClientSet
	DependencyValidator   cloudogu.DependencyValidator
	LocalDoguRegistry     dogu.LocalRegistry
}

// NewManagerSet creates a new ManagerSet.
func NewManagerSet(restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface, config *config.OperatorConfig, configRepos ConfigRepositories, applier cloudogu.Applier, additionalImages map[string]string) (*ManagerSet, error) {
	collectApplier := resource.NewCollectApplier(applier)
	fileExtractor := exec.NewPodFileExtractor(client, restConfig, clientSet)
	commandExecutor := exec.NewCommandExecutor(client, clientSet, clientSet.CoreV1().RESTClient())
	localDoguRegistry := dogu.NewLocalRegistry(clientSet.CoreV1().ConfigMaps(config.Namespace))
	serviceAccountCreator := serviceaccount.NewCreator(configRepos.SensitiveDoguRepository, localDoguRegistry, commandExecutor, client, clientSet, config.Namespace)
	localDoguFetcher := cesregistry.NewLocalDoguFetcher(localDoguRegistry)
	dependencyValidator := dependency.NewCompositeDependencyValidator(config.Version, localDoguRegistry)

	doguRemoteRegistry, err := cesremote.New(config.GetRemoteConfiguration(), config.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	resourceDoguFetcher := cesregistry.NewResourceDoguFetcher(client, doguRemoteRegistry)

	requirementsGenerator := resource.NewRequirementsGenerator(configRepos.DoguConfigRepository)
	hostAliasGenerator := alias.NewHostAliasGenerator(configRepos.GlobalConfigRepository)
	doguResourceGenerator := resource.NewResourceGenerator(client.Scheme(), requirementsGenerator, hostAliasGenerator, additionalImages)

	upserter := resource.NewUpserter(client, doguResourceGenerator)

	doguRegistrator := cesregistry.NewCESDoguRegistrator(client, localDoguRegistry)
	imageRegistry := imageregistry.NewCraneContainerImageRegistry(config.DockerRegistry.Username, config.DockerRegistry.Password)

	return &ManagerSet{
		RestConfig:            restConfig,
		CollectApplier:        collectApplier,
		FileExtractor:         fileExtractor,
		CommandExecutor:       commandExecutor,
		ServiceAccountCreator: serviceAccountCreator,
		LocalDoguRegistry:     localDoguRegistry,
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
