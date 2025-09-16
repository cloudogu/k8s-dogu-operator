package main

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

const testNamespace = "ecosystem"

func Test_newApp(t *testing.T) {
	// given
	restInterfaceMock := newMockRestInterface(t)
	serviceInterfaceMock := newMockServiceInterface(t)
	configMapInterfaceMock := newMockConfigMapInterface(t)
	additionalImagesConfigMap := &corev1.ConfigMap{
		Data: map[string]string{
			config.ChownInitImageConfigmapNameKey:                     "chown-image:1.2.3",
			config.ExporterImageConfigmapNameKey:                      "exporter-image:1.2.3",
			config.AdditionalMountsInitContainerImageConfigmapNameKey: "additional-mounts-init-container-image:1.2.3",
		},
	}
	configMapInterfaceMock.EXPECT().Get(mock.Anything, config.OperatorAdditionalImagesConfigmapName, v1.GetOptions{}).Return(additionalImagesConfigMap, nil)
	secretInterfaceMock := newMockSecretInterface(t)
	pvcInterfaceMock := newMockPvcInterface(t)
	podInterfaceMock := newMockPodInterface(t)
	deploymentInterfaceMock := newMockDeploymentInterface(t)
	coreV1InterfaceMock := newMockCoreV1Interface(t)
	coreV1InterfaceMock.EXPECT().ConfigMaps(testNamespace).Return(configMapInterfaceMock)
	coreV1InterfaceMock.EXPECT().Secrets(testNamespace).Return(secretInterfaceMock)
	coreV1InterfaceMock.EXPECT().Services(testNamespace).Return(serviceInterfaceMock)
	coreV1InterfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(pvcInterfaceMock)
	coreV1InterfaceMock.EXPECT().Pods(testNamespace).Return(podInterfaceMock)
	coreV1InterfaceMock.EXPECT().RESTClient().Return(restInterfaceMock)
	appsV1InterfaceMock := newMockAppsV1Interface(t)
	appsV1InterfaceMock.EXPECT().Deployments(testNamespace).Return(deploymentInterfaceMock)
	kubernetesInterfaceMock := newMockKubernetesInterface(t)
	kubernetesInterfaceMock.EXPECT().CoreV1().Return(coreV1InterfaceMock)
	kubernetesInterfaceMock.EXPECT().AppsV1().Return(appsV1InterfaceMock)

	doguInterfaceMock := newMockDoguInterface(t)
	doguRestartInterfaceMock := newMockDoguRestartInterface(t)
	ecoSystemInterfaceMock := newMockEcoSystemInterface(t)
	ecoSystemInterfaceMock.EXPECT().Dogus(testNamespace).Return(doguInterfaceMock)
	ecoSystemInterfaceMock.EXPECT().DoguRestarts(testNamespace).Return(doguRestartInterfaceMock)

	oldOperatorConfigFn := initfx.NewOperatorConfig
	initfx.NewOperatorConfig = newTestOperatorConfig(t)
	oldKubernetesClientSet := initfx.NewKubernetesClientSet
	initfx.NewKubernetesClientSet = newTestKubernetesInterfaceFn(kubernetesInterfaceMock)
	oldEcoSystemClientSet := initfx.NewEcoSystemClientSet
	initfx.NewEcoSystemClientSet = newTestEcoSystemInterfaceFn(ecoSystemInterfaceMock)
	oldGetRestConfig := ctrl.GetConfig
	ctrl.GetConfig = newTestGetConfig()
	t.Cleanup(func() {
		initfx.NewOperatorConfig = oldOperatorConfigFn
		initfx.NewKubernetesClientSet = oldKubernetesClientSet
		initfx.NewEcoSystemClientSet = oldEcoSystemClientSet
		ctrl.GetConfig = oldGetRestConfig
	})

	// when
	newApp := newApp()

	// then
	assert.NoError(t, newApp.Err())
}

func newTestGetConfig() func() (*rest.Config, error) {
	return func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}
}

func newTestKubernetesInterfaceFn(p kubernetes.Interface) func(c *rest.Config) (kubernetes.Interface, error) {
	return func(c *rest.Config) (kubernetes.Interface, error) {
		return p, nil
	}
}

func newTestEcoSystemInterfaceFn(v2Interface doguClient.EcoSystemV2Interface) func(c *rest.Config) (doguClient.EcoSystemV2Interface, error) {
	return func(c *rest.Config) (doguClient.EcoSystemV2Interface, error) {
		return v2Interface, nil
	}
}

func newTestOperatorConfig(t *testing.T) func(version config.Version) (*config.OperatorConfig, error) {
	return func(version config.Version) (*config.OperatorConfig, error) {
		parsed, err := core.ParseVersion(string(version))
		assert.NoError(t, err)

		return &config.OperatorConfig{
			Namespace: testNamespace,
			DoguRegistry: config.DoguRegistryData{
				Endpoint:  "myEndpoint",
				Username:  "myUsername",
				Password:  "myPassword",
				URLSchema: "default",
			},
			Version:                &parsed,
			NetworkPoliciesEnabled: true,
		}, nil
	}
}
