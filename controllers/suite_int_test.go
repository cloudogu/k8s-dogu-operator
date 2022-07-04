//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	_ "embed"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var k8sClient client.Client
var testEnv *envtest.Environment
var cancel context.CancelFunc

// Used in other integration tests
var ImageRegistryMock mocks.ImageRegistry

// Used in other integration tests
var DoguRemoteRegistryMock cesremotemocks.Registry

// Used in other integration tests
var EtcdDoguRegistry cesmocks.DoguRegistry

const TimeoutInterval = time.Second * 10
const PollingInterval = time.Second * 1

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	// We need to ensure that the development stage flag is not passed by our makefiles to prevent the dogu operator
	// from running in the developing mode. The developing mode changes some operator behaviour. Our integration test
	// aim to test the production functionality of the operator.
	err := os.Unsetenv(config.StageEnvironmentVariable)
	Expect(err).NotTo(HaveOccurred())
	err = os.Setenv(config.StageEnvironmentVariable, config.StageProduction)
	Expect(err).NotTo(HaveOccurred())
	config.Stage = config.StageProduction

	logf.SetLogger(logrusr.New(logrus.New()))

	var ctx context.Context
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = k8sv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	resourceGenerator := resource.NewResourceGenerator(k8sManager.GetScheme())

	doguConfigurationContext := &cesmocks.ConfigurationContext{}
	doguConfigurationContext.On("Set", mock.Anything, mock.Anything).Return(nil)
	doguConfigurationContext.On("RemoveAll", mock.Anything).Return(nil)

	globalConfigurationContext := &cesmocks.ConfigurationContext{}
	globalConfigurationContext.On("Get", "key_provider").Return("", nil)

	CesRegistryMock := cesmocks.Registry{}
	CesRegistryMock.On("DoguRegistry").Return(&EtcdDoguRegistry)
	CesRegistryMock.On("DoguConfig", mock.Anything).Return(doguConfigurationContext)
	CesRegistryMock.On("GlobalConfig").Return(globalConfigurationContext)

	version, err := core.ParseVersion("0.0.0")
	Expect(err).ToNot(HaveOccurred())

	dependencyValidator := dependency.NewCompositeDependencyValidator(&version, &EtcdDoguRegistry)
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	serviceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	serviceAccountRemover := &mocks.ServiceAccountRemover{}
	serviceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	doguSecretHandler := &mocks.DoguSecretsHandler{}
	doguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)

	doguRegistrator := controllers.NewCESDoguRegistrator(k8sManager.GetClient(), &CesRegistryMock, resourceGenerator)

	yamlResult := make(map[string]string, 0)
	fileExtract := &mockFileExtractor{}
	fileExtract.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
	applyClient := &mockApplier{}
	applyClient.On("Apply", mock.Anything, mock.Anything).Return(nil)

	doguManager := &controllers.DoguManager{
		Client:                k8sManager.GetClient(),
		Scheme:                k8sManager.GetScheme(),
		ResourceGenerator:     resourceGenerator,
		DoguRemoteRegistry:    &DoguRemoteRegistryMock,
		DoguLocalRegistry:     &EtcdDoguRegistry,
		ImageRegistry:         &ImageRegistryMock,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		ServiceAccountRemover: serviceAccountRemover,
		DoguSecretHandler:     doguSecretHandler,
		Applier:               applyClient,
		FileExtractor:         fileExtract,
	}

	err = controllers.NewDoguReconciler(k8sManager.GetClient(), k8sManager.GetScheme(), doguManager).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())
}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
