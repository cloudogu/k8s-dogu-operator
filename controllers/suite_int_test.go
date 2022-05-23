//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	_ "embed"
	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"

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
var DoguRegistryMock mocks.DoguRegistry

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
	doguConfigurationContext.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
	doguConfigurationContext.Mock.On("RemoveAll", mock.Anything).Return(nil)

	globalConfigurationContext := &cesmocks.ConfigurationContext{}
	globalConfigurationContext.Mock.On("Get", "key_provider").Return("", nil)

	CesRegistryMock := cesmocks.Registry{}
	CesRegistryMock.Mock.On("DoguRegistry").Return(&EtcdDoguRegistry)
	CesRegistryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigurationContext)
	CesRegistryMock.Mock.On("GlobalConfig").Return(globalConfigurationContext)

	version, err := core.ParseVersion("0.0.0")
	Expect(err).ToNot(HaveOccurred())

	dependencyValidator := dependency.NewCompositeDependencyValidator(&version, &EtcdDoguRegistry)
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	serviceAccountCreator.Mock.On("CreateServiceAccounts", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	doguSecretHandler := &mocks.DoguSecretsHandler{}
	doguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)

	doguRegistrator := controllers.NewCESDoguRegistrator(k8sManager.GetClient(), &CesRegistryMock, resourceGenerator)
	doguManager := controllers.DoguManager{
		Client:                k8sManager.GetClient(),
		Scheme:                k8sManager.GetScheme(),
		ResourceGenerator:     resourceGenerator,
		DoguRegistry:          &DoguRegistryMock,
		ImageRegistry:         &ImageRegistryMock,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		DoguSecretHandler:     doguSecretHandler,
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
