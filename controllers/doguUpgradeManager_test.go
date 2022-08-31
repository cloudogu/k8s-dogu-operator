package controllers

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type doguUpgradeManagerWithMocks struct {
	*doguUpgradeManager
	doguRemoteRegistryMock    *cesremotemocks.Registry
	doguLocalRegistryMock     *cesmocks.DoguRegistry
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	dependencyValidatorMock   *mocks.DependencyValidator
	serviceAccountCreatorMock *mocks.ServiceAccountCreator
	applierMock               *mocks.Applier
	client                    client.WithWatch
}

func (dum *doguUpgradeManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		dum.doguRemoteRegistryMock,
		dum.doguLocalRegistryMock,
		dum.imageRegistryMock,
		dum.doguRegistratorMock,
		dum.dependencyValidatorMock,
		dum.serviceAccountCreatorMock,
		dum.applierMock,
	)
}

func getDoguUpgradeManagerWithMocks(scheme *runtime.Scheme) doguUpgradeManagerWithMocks {
	mockK8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	mockDoguRemoteRegistry := &cesremotemocks.Registry{}
	mockDoguLocalRegistry := &cesmocks.DoguRegistry{}
	mockImageRegistry := &mocks.ImageRegistry{}
	mockDoguRegistrator := &mocks.DoguRegistrator{}
	mockDependencyValidator := &mocks.DependencyValidator{}
	mockServiceAccountCreator := &mocks.ServiceAccountCreator{}
	mockedApplier := &mocks.Applier{}

	sut := &doguUpgradeManager{
		client:                mockK8sClient,
		scheme:                scheme,
		doguRemoteRegistry:    mockDoguRemoteRegistry,
		doguLocalRegistry:     mockDoguLocalRegistry,
		imageRegistry:         mockImageRegistry,
		doguRegistrator:       mockDoguRegistrator,
		dependencyValidator:   mockDependencyValidator,
		serviceAccountCreator: mockServiceAccountCreator,
		applier:               mockedApplier,
	}

	return doguUpgradeManagerWithMocks{
		doguUpgradeManager:        sut,
		client:                    mockK8sClient,
		doguRemoteRegistryMock:    mockDoguRemoteRegistry,
		doguLocalRegistryMock:     mockDoguLocalRegistry,
		imageRegistryMock:         mockImageRegistry,
		doguRegistratorMock:       mockDoguRegistrator,
		dependencyValidatorMock:   mockDependencyValidator,
		serviceAccountCreatorMock: mockServiceAccountCreator,
		applierMock:               mockedApplier,
	}
}

func getDoguUpgradeManagerTestData(t *testing.T) (*k8sv1.Dogu, *core.Dogu, *corev1.ConfigMap, *imagev1.ConfigFile) {
	ldapCr := readTestDataLdapCr(t)
	ldapDogu := readTestDataLdapDogu(t)
	ldapDoguDescriptor := readTestDataLdapDescriptor(t)
	imageConfig := readTestDataImageConfig(t)
	return ldapCr, ldapDogu, ldapDoguDescriptor, imageConfig
}

func TestNewDoguUpgradeManager(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = func() *rest.Config {
		return &rest.Config{}
	}

	t.Run("fail when no valid kube config was found", func(t *testing.T) {
		// given

		// override default controller method to return a config that fail the client creation
		oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
		defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{ExecProvider: &api.ExecConfig{}, AuthProvider: &api.AuthProviderConfig{}}
		}

		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"

		// when
		doguManager, err := NewDoguUpgradeManager(nil, operatorConfig, nil)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})

	t.Run("should implement upgradeManager", func(t *testing.T) {
		myClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		actual, err := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry)

		// then
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Implements(t, (*upgradeManager)(nil), actual)
		cesRegistry.AssertExpectations(t)
	})
}
