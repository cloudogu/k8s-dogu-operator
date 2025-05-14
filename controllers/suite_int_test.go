//go:build k8s_integration

package controllers

import (
	"context"
	_ "embed"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dataseed"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	registryRepo "github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"testing"
	"time"

	"github.com/bombsimon/logrusr/v2"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var ecosystemClientSet *doguClient.EcoSystemV2Client
var k8sClientSet ClientSet
var testEnv *envtest.Environment
var cancel context.CancelFunc

// Used in other integration tests
var (
	ImageRegistryMock      *mockImageRegistry
	CommandExecutorMock    *mockCommandExecutor
	DoguRemoteRegistryMock *mockRemoteDoguDescriptorRepository
	k8sClient              K8sClient
	DoguInterfaceMock      *mockDoguInterface
)

const TimeoutInterval = time.Second * 10
const PollingInterval = time.Second * 1

var oldGetConfig func() (*rest.Config, error)
var oldGetConfigOrDie func() *rest.Config
var oldCtrlBuilder func(m manager.Manager) *ctrl.Builder

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controller Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	// We need to ensure that the development stage flag is not passed by our makefiles to prevent the dogu operator
	// from running in the developing mode. The developing mode changes some operator behaviour. Our integration test
	// aim to test the production functionality of the operator.
	err := os.Unsetenv(config.StageEnvironmentVariable)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	err = os.Setenv(config.StageEnvironmentVariable, config.StageProduction)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	config.Stage = config.StageProduction

	CommandExecutorMock = &mockCommandExecutor{}
	logf.SetLogger(logrusr.New(logrus.New()))

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "vendor", "github.com", "cloudogu", "k8s-dogu-lib", "v2", "api", "v2")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	oldGetConfig = ctrl.GetConfig
	ctrl.GetConfig = func() (*rest.Config, error) {
		return cfg, nil
	}

	oldGetConfigOrDie = ctrl.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return cfg
	}

	// override default controller-builder and add skipNameValidation for tests
	oldCtrlBuilder = ctrl.NewControllerManagedBy
	ctrl.NewControllerManagedBy = func(m manager.Manager) *ctrl.Builder {
		builder := oldCtrlBuilder(m)
		skipNameValidation := true
		builder.WithOptions(controller.Options{SkipNameValidation: &skipNameValidation})

		return builder
	}

	err = doguv2.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// +kubebuilder:scaffold:scheme
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClient = k8sManager.GetClient()
	gomega.Expect(k8sClient).ToNot(gomega.BeNil())

	ecosystemClientSet, err = doguClient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClientSet, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	DoguRemoteRegistryMock = &mockRemoteDoguDescriptorRepository{}
	ImageRegistryMock = &mockImageRegistry{}
	DoguInterfaceMock = &mockDoguInterface{}

	requirementsGen := &mockRequirementsGenerator{}
	requirementsGen.EXPECT().Generate(mock.Anything, mock.Anything).Return(v1.ResourceRequirements{}, nil)
	hostAliasGeneratorMock := &mockHostAliasGenerator{}
	hostAliasGeneratorMock.On("Generate", mock.Anything).Return(nil, nil)

	securityValidator := &security.Validator{}
	securityGenerator := &resource.SecurityContextGenerator{}

	doguDataSeedValidator := &dataseed.Validator{}

	additionalImages := map[string]string{config.ChownInitImageConfigmapNameKey: "image:tag", config.DataSeederImageConfigmapNameKey: "test:tag"}
	resourceGenerator := resource.NewResourceGenerator(k8sManager.GetScheme(), requirementsGen, hostAliasGeneratorMock, securityGenerator, additionalImages)

	version, err := core.ParseVersion("0.0.0")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	doguVersionRegistry := dogu.NewDoguVersionRegistry(k8sClientSet.CoreV1().ConfigMaps(testNamespace))
	localDoguDescriptorRepository := dogu.NewLocalDoguDescriptorRepository(k8sClientSet.CoreV1().ConfigMaps(testNamespace))
	localDoguFetcher := cesregistry.NewLocalDoguFetcher(doguVersionRegistry, localDoguDescriptorRepository)

	dependencyValidator := dependency.NewCompositeDependencyValidator(&version, localDoguFetcher)
	serviceAccountCreator := &mockServiceAccountCreator{}
	serviceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	serviceAccountRemover := &mockServiceAccountRemover{}
	serviceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	doguRegistrator := cesregistry.NewCESDoguRegistrator(doguVersionRegistry, localDoguDescriptorRepository)

	yamlResult := make(map[string]string)
	fileExtract := &mockFileExtractor{}
	fileExtract.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
	applyClient := &mockApplier{}
	applyClient.On("Apply", mock.Anything, mock.Anything).Return(nil)

	eventRecorder := k8sManager.GetEventRecorderFor("k8s-dogu-operator")
	upserter := resource.NewUpserter(k8sClient, resourceGenerator, true)
	collectApplier := resource.NewCollectApplier(applyClient)

	remoteDoguFetcher := cesregistry.NewResourceDoguFetcher(k8sClient, DoguRemoteRegistryMock)
	execPodFactory := exec.NewExecPodFactory(k8sClient, cfg, CommandExecutorMock)

	sensitiveConfigRepo := registryRepo.NewSensitiveDoguConfigRepository(k8sClientSet.CoreV1().Secrets(testNamespace))
	doguConfigRepo := registryRepo.NewDoguConfigRepository(k8sClientSet.CoreV1().ConfigMaps(testNamespace))

	installManager := &doguInstallManager{
		client:                  k8sClient,
		ecosystemClient:         ecosystemClientSet,
		recorder:                eventRecorder,
		resourceUpserter:        upserter,
		resourceDoguFetcher:     remoteDoguFetcher,
		imageRegistry:           ImageRegistryMock,
		doguRegistrator:         doguRegistrator,
		dependencyValidator:     dependencyValidator,
		serviceAccountCreator:   serviceAccountCreator,
		collectApplier:          collectApplier,
		fileExtractor:           fileExtract,
		localDoguFetcher:        localDoguFetcher,
		execPodFactory:          execPodFactory,
		sensitiveDoguRepository: sensitiveConfigRepo,
		doguConfigRepository:    doguConfigRepo,
		securityValidator:       securityValidator,
		doguDataSeedValidator:   doguDataSeedValidator,
	}

	deleteManager := &doguDeleteManager{
		client:                  k8sClient,
		doguRegistrator:         doguRegistrator,
		serviceAccountRemover:   serviceAccountRemover,
		localDoguFetcher:        localDoguFetcher,
		doguConfigRepository:    doguConfigRepo,
		sensitiveDoguRepository: sensitiveConfigRepo,
	}

	volumeManager := &doguVolumeManager{
		client:        k8sClient,
		eventRecorder: eventRecorder,
	}

	ingressAnnotationManager := &doguAdditionalIngressAnnotationsManager{
		client:        k8sClient,
		eventRecorder: eventRecorder,
	}

	doguHealthChecker := health.NewDoguChecker(ecosystemClientSet, localDoguFetcher)
	upgradePremiseChecker := upgrade.NewPremisesChecker(dependencyValidator, doguHealthChecker, doguHealthChecker, securityValidator, doguDataSeedValidator)

	mgrSet := &util.ManagerSet{
		RestConfig:                     ctrl.GetConfigOrDie(),
		ImageRegistry:                  ImageRegistryMock,
		ServiceAccountCreator:          serviceAccountCreator,
		FileExtractor:                  fileExtract,
		CollectApplier:                 collectApplier,
		SecurityValidator:              securityValidator,
		CommandExecutor:                CommandExecutorMock,
		ResourceUpserter:               upserter,
		DoguRegistrator:                doguRegistrator,
		LocalDoguFetcher:               localDoguFetcher,
		DoguResourceGenerator:          resourceGenerator,
		ResourceDoguFetcher:            remoteDoguFetcher,
		DoguDataSeedValidator:          doguDataSeedValidator,
		AdditionalImages:               additionalImages,
		RequirementsGenerator:          requirementsGen,
		DoguDataSeedContainerGenerator: resourceGenerator,
	}

	upgradeExecutor := upgrade.NewUpgradeExecutor(k8sClient, mgrSet, eventRecorder, ecosystemClientSet)

	upgradeManager := &doguUpgradeManager{
		client:              k8sClient,
		ecosystemClient:     ecosystemClientSet,
		eventRecorder:       eventRecorder,
		premisesChecker:     upgradePremiseChecker,
		localDoguFetcher:    localDoguFetcher,
		resourceDoguFetcher: remoteDoguFetcher,
		upgradeExecutor:     upgradeExecutor,
	}

	supportManager := &doguSupportManager{
		client:                       k8sManager.GetClient(),
		doguFetcher:                  localDoguFetcher,
		podTemplateResourceGenerator: resourceGenerator,
		eventRecorder:                eventRecorder,
	}

	securityManager := &DoguSecurityContextManager{
		localDoguFetcher:  localDoguFetcher,
		resourceUpserter:  upserter,
		securityValidator: securityValidator,
		recorder:          eventRecorder,
	}

	dataSeedManager := NewDoguDataSeedManager(k8sClientSet.AppsV1().Deployments(testNamespace), mgrSet)

	doguManager := &DoguManager{
		scheme:                    k8sManager.GetScheme(),
		installManager:            installManager,
		deleteManager:             deleteManager,
		supportManager:            supportManager,
		upgradeManager:            upgradeManager,
		recorder:                  eventRecorder,
		volumeManager:             volumeManager,
		ingressAnnotationsManager: ingressAnnotationManager,
		securityContextManager:    securityManager,
		dataSeedManager:           dataSeedManager,
	}

	doguReconciler, err := NewDoguReconciler(k8sClient, DoguInterfaceMock, doguManager, eventRecorder, testNamespace, localDoguFetcher)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = doguReconciler.SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	updater := health.NewDoguStatusUpdater(ecosystemClientSet, eventRecorder, k8sClientSet)
	deploymentReconciler := NewDeploymentReconciler(k8sClientSet, &health.AvailabilityChecker{}, updater, localDoguFetcher)

	err = deploymentReconciler.SetupWithManager(k8sManager)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}()
}, 60)

var _ = ginkgo.AfterSuite(func() {
	cancel()
	ginkgo.By("tearing down the test environment")
	err := testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ctrl.GetConfig = oldGetConfig
	ctrl.GetConfigOrDie = oldGetConfigOrDie
	ctrl.NewControllerManagedBy = oldCtrlBuilder
})
