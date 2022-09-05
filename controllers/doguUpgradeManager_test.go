package controllers

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

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

func TestNewDoguUpgradeManager(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

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

func Test_checkUpgradeability(t *testing.T) {
	t.Run("should succeed for dogus when forceUpgrade is off and remote dogu has a higher version", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Version = "2.4.48-5"

		// when
		err := checkUpgradeability(nil, localDogu, remoteDogu, false)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed for dogus when forceUpgrade is on but would originally fail because of versions or names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/ldap"

		// when
		err := checkUpgradeability(nil, localDogu, remoteDogu, true)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = remoteDogu.GetNamespace() + "/test"
		// when
		err := checkUpgradeability(nil, localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "upgrade-ability check failed: dogus must have the same name (ldap=test)", err.Error())
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/ldap"
		// when
		err := checkUpgradeability(nil, localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "upgrade-ability check failed: dogus must have the same namespace (official=different-ns)", err.Error())
	})
}

func Test_doguUpgradeManager_checkDoguHealth(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig
	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		testDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "testServiceAccount",
					},
				},
			},
		}
		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(testDeployment.GroupVersionKind(), &appsv1.Deployment{})
		myClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry)

		// when
		err := sut.doguHealthChecker.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.NoError(t, err)
		cesRegistry.AssertExpectations(t)
		doguRegistry.AssertExpectations(t)
	})
	t.Run("should fail because of unready replicas", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		testDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "testServiceAccount",
					},
				},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 0}, // trigger failure
		}
		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(testDeployment.GroupVersionKind(), &appsv1.Deployment{})
		myClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry)

		// when
		err := sut.doguHealthChecker.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogu appears unhealthy (expected: 1, ready: 0)", err.Error())
		cesRegistry.AssertExpectations(t)
		doguRegistry.AssertExpectations(t)
	})
}

func Test_doguUpgradeManager_checkDependencyDogusHealthy(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed when all dogu dependencies are in a healthy state", func(t *testing.T) {
		// when

		// then

	})
	t.Run("should fail when at least one dependency dogus is unhealthy", func(t *testing.T) {
		redmineCr := readTestDataRedmineCr(t)

		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		dependentDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "redmine",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 1},
		}
		dependencyDeployment := &appsv1.Deployment{
			TypeMeta: deploymentTypeMeta,
			ObjectMeta: metav1.ObjectMeta{
				Name:      "postgresql",
				Namespace: testNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmpty"}},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 0},
		}

		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(dependencyDeployment.GroupVersionKind(), &appsv1.Deployment{})
		myClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(dependentDeployment, dependencyDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry)

		// when
		err := sut.checkDependencyDogusHealthy(context.TODO(), redmineCr, nil)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogu appears unhealthy (expected: 1, ready: 0)", err.Error())
		cesRegistry.AssertExpectations(t)
		doguRegistry.AssertExpectations(t)
	})
}

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
}
