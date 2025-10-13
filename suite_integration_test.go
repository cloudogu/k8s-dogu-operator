//go:build k8s_integration

package main

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bombsimon/logrusr/v2"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx/fxtest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
var ecosystemClientSet *doguClient.EcoSystemV2Client
var k8sClientSet controllers.ClientSet
var testEnv *envtest.Environment

var testCtx = context.Background()

// Used in other integration tests
var (
	ImageRegistryMock                  *mockImageRegistry
	CommandExecutorMock                *mockCommandExecutor
	RemoteDoguDescriptorRepositoryMock *mockRemoteDoguDescriptorRepository
	k8sClient                          controllers.K8sClient
	fxApp                              *fxtest.App
)

const TimeoutInterval = time.Second * 20
const PollingInterval = time.Second * 1

var (
	oldGetConfig                         func() (*rest.Config, error)
	oldGetConfigOrDie                    func() *rest.Config
	oldCtrlBuilder                       func(m manager.Manager) *ctrl.Builder
	oldNewCommandExecutor                func(cli client.Client, restConfig *rest.Config, clientSet kubernetes.Interface, coreV1RestClient rest.Interface) exec.CommandExecutor
	oldNewRemoteDoguDescriptorRepository func(operatorConfig *config.OperatorConfig) (dogu.RemoteDoguDescriptorRepository, error)
	oldNewImageRegistry                  func() imageregistry.ImageRegistry
	oldGetArgs                           func() initfx.Args
)

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controller Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("setting env vars")
	err := os.Setenv("LOG_LEVEL", "debug")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("NAMESPACE", "ecosystem")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("DOGU_REGISTRY_ENDPOINT", "https://dogu.cloudogu.com")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("REQUEUE_TIME_FOR_DOGU_RESOURCE_IN_NANOSECONDS", "1")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("DOGU_REGISTRY_USERNAME", "ecosystem")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("DOGU_REGISTRY_PASSWORD", "ecosystem")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = os.Setenv("NETWORK_POLICIES_ENABLED", "true")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = os.Setenv(config.StageEnvironmentVariable, config.StageProduction)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	config.Stage = config.StageProduction

	ginkgo.By("bootstrapping test environment")
	logf.SetLogger(logrusr.New(logrus.New()))
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("vendor", "github.com", "cloudogu", "k8s-dogu-lib", "v2", "api", "v2")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	ginkgo.By("creating mocks")
	CommandExecutorMock = &mockCommandExecutor{}
	RemoteDoguDescriptorRepositoryMock = &mockRemoteDoguDescriptorRepository{}
	ImageRegistryMock = &mockImageRegistry{}

	ginkgo.By("overriding functions")
	oldNewCommandExecutor = initfx.NewCommandExecutor
	initfx.NewCommandExecutor = func(client.Client, *rest.Config, kubernetes.Interface, rest.Interface) exec.CommandExecutor {
		return CommandExecutorMock
	}

	oldNewRemoteDoguDescriptorRepository = initfx.NewRemoteDoguDescriptorRepository
	initfx.NewRemoteDoguDescriptorRepository = func(operatorConfig *config.OperatorConfig) (dogu.RemoteDoguDescriptorRepository, error) {
		return RemoteDoguDescriptorRepositoryMock, nil
	}

	oldNewImageRegistry = initfx.NewImageRegistry
	initfx.NewImageRegistry = func() imageregistry.ImageRegistry {
		return ImageRegistryMock
	}

	oldGetArgs = initfx.GetArgs
	initfx.GetArgs = func() initfx.Args {
		return initfx.Args{"k8s-dogu-operator"}
	}

	oldGetConfig = ctrl.GetConfig
	ctrl.GetConfig = func() (*rest.Config, error) {
		return cfg, nil
	}

	oldCtrlBuilder = ctrl.NewControllerManagedBy
	ctrl.NewControllerManagedBy = func(m manager.Manager) *ctrl.Builder {
		builder := oldCtrlBuilder(m)
		skipNameValidation := true
		builder.WithOptions(controller.Options{SkipNameValidation: &skipNameValidation})

		return builder
	}

	ginkgo.By("creating clients")
	ecosystemClientSet, err = doguClient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClientSet, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	ginkgo.By("creating operator config")
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Namespace: testNamespace}}
	err = k8sClient.Create(testCtx, namespace)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	additionalImagesCm := readConfigMap(ginkgo.GinkgoT(), additionalImagesCmBytes)
	_, err = k8sClientSet.CoreV1().ConfigMaps(testNamespace).Create(testCtx, additionalImagesCm, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	globalConfig := readConfigMap(ginkgo.GinkgoT(), globalConfigBytes)
	_, err = k8sClientSet.CoreV1().ConfigMaps(testNamespace).Create(testCtx, globalConfig, metav1.CreateOptions{})
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	ginkgo.By("starting application")
	go func() {
		defer ginkgo.GinkgoRecover()
		fxApp = fxtest.New(ginkgo.GinkgoT(), options()...).RequireStart()
	}()
}, 60)

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("tearing down the test environment")
	fxApp.RequireStop()
	err := testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	initfx.NewCommandExecutor = oldNewCommandExecutor
	initfx.NewRemoteDoguDescriptorRepository = oldNewRemoteDoguDescriptorRepository
	initfx.NewImageRegistry = oldNewImageRegistry
	initfx.GetArgs = oldGetArgs

	ctrl.GetConfig = oldGetConfig
	ctrl.GetConfigOrDie = oldGetConfigOrDie
	ctrl.NewControllerManagedBy = oldCtrlBuilder
})
