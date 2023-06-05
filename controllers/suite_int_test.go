//go:build k8s_integration
// +build k8s_integration

package controllers

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bombsimon/logrusr/v2"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var k8sClientSet *ecoSystem.EcoSystemV1Alpha1Client
var testEnv *envtest.Environment
var cancel context.CancelFunc

// Used in other integration tests
var (
	ImageRegistryMock      *mocks.ImageRegistry
	CommandExecutor        *mocks.CommandExecutor
	DoguRemoteRegistryMock *extMocks.RemoteRegistry
	EtcdDoguRegistry       *extMocks.DoguRegistry
	k8sClient              thirdParty.K8sClient
)

const TimeoutInterval = time.Second * 10
const PollingInterval = time.Second * 1

var oldGetConfig func() (*rest.Config, error)
var oldGetConfigOrDie func() *rest.Config

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

	CommandExecutor = &mocks.CommandExecutor{}
	logf.SetLogger(logrusr.New(logrus.New()))

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.TODO())

	ginkgo.By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
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

	err = k8sv1.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// +kubebuilder:scaffold:scheme
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClient = k8sManager.GetClient()
	gomega.Expect(k8sClient).ToNot(gomega.BeNil())

	k8sClientSet, err = ecoSystem.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	DoguRemoteRegistryMock = &extMocks.RemoteRegistry{}
	EtcdDoguRegistry = &extMocks.DoguRegistry{}
	ImageRegistryMock = &mocks.ImageRegistry{}

	doguConfigurationContext := &cesmocks.ConfigurationContext{}
	doguConfigurationContext.On("Set", mock.Anything, mock.Anything).Return(nil)
	doguConfigurationContext.On("RemoveAll", mock.Anything).Return(nil)
	doguConfigurationContext.On("Get", "/pod_limit/cpu").Return("1", nil)
	doguConfigurationContext.On("Get", "/pod_limit/memory").Return("1", nil)
	doguConfigurationContext.On("Get", "/pod_limit/ephemeral_storage").Return("1", nil)

	globalConfigurationContext := &cesmocks.ConfigurationContext{}
	globalConfigurationContext.On("Get", "key_provider").Return("", nil)
	globalConfigurationContext.On("Get", "fqdn").Return("", nil)
	globalConfigurationContext.On("Get", "k8s/use_internal_ip").Return("false", nil)
	globalConfigurationContext.On("GetAll").Return(map[string]string{}, nil)

	CesRegistryMock := &cesmocks.Registry{}
	CesRegistryMock.On("DoguRegistry").Return(EtcdDoguRegistry)
	CesRegistryMock.On("DoguConfig", mock.Anything).Return(doguConfigurationContext)
	CesRegistryMock.On("GlobalConfig").Return(globalConfigurationContext)

	limitPatcher := &mocks.LimitPatcher{}
	doguLimits := &mocks.DoguLimits{}
	limitPatcher.On("RetrievePodLimits", mock.Anything).Return(doguLimits, nil)
	limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
	hostAliasGeneratorMock := &extMocks.HostAliasGenerator{}
	hostAliasGeneratorMock.On("Generate").Return(nil, nil)
	resourceGenerator := resource.NewResourceGenerator(k8sManager.GetScheme(), limitPatcher, hostAliasGeneratorMock)

	version, err := core.ParseVersion("0.0.0")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	dependencyValidator := dependency.NewCompositeDependencyValidator(&version, EtcdDoguRegistry)
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	serviceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	serviceAccountRemover := &mocks.ServiceAccountRemover{}
	serviceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	doguSecretHandler := &mocks.DoguSecretHandler{}
	doguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)

	doguRegistrator := cesregistry.NewCESDoguRegistrator(k8sClient, CesRegistryMock, resourceGenerator)

	yamlResult := make(map[string]string, 0)
	fileExtract := &mocks.FileExtractor{}
	fileExtract.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
	applyClient := &mocks.Applier{}
	applyClient.On("Apply", mock.Anything, mock.Anything).Return(nil)

	eventRecorder := k8sManager.GetEventRecorderFor("k8s-dogu-operator")
	upserter := resource.NewUpserter(k8sClient, limitPatcher, hostAliasGeneratorMock)
	collectApplier := resource.NewCollectApplier(applyClient)

	localDoguFetcher := cesregistry.NewLocalDoguFetcher(EtcdDoguRegistry)
	remoteDoguFetcher := cesregistry.NewResourceDoguFetcher(k8sClient, DoguRemoteRegistryMock)
	execPodFactory := exec.NewExecPodFactory(k8sClient, cfg, CommandExecutor)
	exposedPortRemover := resource.NewDoguExposedPortHandler(k8sClient)

	installManager := &doguInstallManager{
		client:                k8sClient,
		recorder:              eventRecorder,
		resourceUpserter:      upserter,
		doguRemoteRegistry:    DoguRemoteRegistryMock,
		doguLocalRegistry:     EtcdDoguRegistry,
		resourceDoguFetcher:   remoteDoguFetcher,
		imageRegistry:         ImageRegistryMock,
		doguRegistrator:       doguRegistrator,
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		doguSecretHandler:     doguSecretHandler,
		collectApplier:        collectApplier,
		fileExtractor:         fileExtract,
		localDoguFetcher:      localDoguFetcher,
		execPodFactory:        execPodFactory,
	}

	deleteManager := &doguDeleteManager{
		client:                k8sClient,
		imageRegistry:         ImageRegistryMock,
		doguRegistrator:       doguRegistrator,
		serviceAccountRemover: serviceAccountRemover,
		doguSecretHandler:     doguSecretHandler,
		localDoguFetcher:      localDoguFetcher,
		exposedPortRemover:    exposedPortRemover,
	}

	volumeManager := &doguVolumeManager{
		client:        k8sClient,
		eventRecorder: eventRecorder,
	}

	ingressAnnotationManager := &doguAdditionalIngressAnnotationsManager{
		client:        k8sClient,
		eventRecorder: eventRecorder,
	}

	doguHealthChecker := health.NewDoguChecker(k8sClient, localDoguFetcher)
	upgradePremiseChecker := upgrade.NewPremisesChecker(dependencyValidator, doguHealthChecker, doguHealthChecker)
	upgradeExecutor := upgrade.NewUpgradeExecutor(k8sClient, cfg, CommandExecutor, eventRecorder, ImageRegistryMock, collectApplier, fileExtract, serviceAccountCreator, CesRegistryMock)

	upgradeManager := &doguUpgradeManager{
		client:              k8sClient,
		eventRecorder:       eventRecorder,
		premisesChecker:     upgradePremiseChecker,
		localDoguFetcher:    localDoguFetcher,
		resourceDoguFetcher: remoteDoguFetcher,
		upgradeExecutor:     upgradeExecutor,
	}

	supportManager := &doguSupportManager{
		client:            k8sManager.GetClient(),
		scheme:            k8sManager.GetScheme(),
		doguRegistry:      EtcdDoguRegistry,
		resourceGenerator: resourceGenerator,
		eventRecorder:     eventRecorder,
	}

	doguManager := &DoguManager{
		scheme:                    k8sManager.GetScheme(),
		installManager:            installManager,
		deleteManager:             deleteManager,
		supportManager:            supportManager,
		upgradeManager:            upgradeManager,
		recorder:                  eventRecorder,
		volumeManager:             volumeManager,
		ingressAnnotationsManager: ingressAnnotationManager,
	}

	reconciler, err := NewDoguReconciler(k8sClient, doguManager, eventRecorder, testNamespace, EtcdDoguRegistry)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = reconciler.SetupWithManager(k8sManager)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

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
})
