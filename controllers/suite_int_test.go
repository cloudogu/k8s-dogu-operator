//go:build k8s_integration
// +build k8s_integration

package controllers_test

import (
	"context"
	_ "embed"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var k8sClient client.Client
var testEnv *envtest.Environment
var cancel context.CancelFunc

// Used in controller integration test
var ImageRegistryMock mocks.ImageRegistry

// Used in controller integration test
var DoguRegistryMock mocks.DoguRegistry

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

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

	resourceGenerator := controllers.NewResourceGenerator(k8sManager.GetScheme())
	CesRegistryMock := cesmocks.Registry{}
	registry := &cesmocks.DoguRegistry{}
	EtcdDoguRegistry := &cesmocks.DoguRegistry{}
	doguConfigurationContext := &cesmocks.ConfigurationContext{}
	doguConfigurationContext.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
	doguConfigurationContext.Mock.On("RemoveAll", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("Register", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("Unregister", mock.Anything).Return(nil)
	EtcdDoguRegistry.Mock.On("Enable", mock.Anything).Return(nil)
	CesRegistryMock.Mock.On("DoguRegistry").Return(EtcdDoguRegistry)
	CesRegistryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigurationContext)

	version, err := core.ParseVersion("0.0.0")
	Expect(err).ToNot(HaveOccurred())

	doguRegistrator := controllers.NewCESDoguRegistrator(k8sManager.GetClient(), &CesRegistryMock, resourceGenerator)
	doguManager := controllers.NewDoguManager(&version, k8sManager.GetClient(), k8sManager.GetScheme(), resourceGenerator, &DoguRegistryMock, &ImageRegistryMock, doguRegistrator, registry)

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
