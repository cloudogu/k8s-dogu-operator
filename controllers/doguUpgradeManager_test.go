package controllers

import (
	"context"
	"fmt"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/coreos/etcd/client"
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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const defaultNamespace = ""

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

var registryKeyNotFoundTestErr = fmt.Errorf("oh no: %w", &client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"})

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
}

func createReadyDeployment(doguName string) *appsv1.Deployment {
	return createDeployment(doguName, 1, 1)
}

func createDeployment(doguName string, replicas, replicasReady int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: defaultNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: replicas, ReadyReplicas: replicasReady},
	}
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
		doguManager, err := NewDoguUpgradeManager(nil, operatorConfig, nil, nil)

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
		actual, err := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)

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
		err := checkUpgradeability(localDogu, remoteDogu, false)

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed for dogus when forceUpgrade is on but would originally fail because of versions or names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/ldap"

		// when
		err := checkUpgradeability(localDogu, remoteDogu, true)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = remoteDogu.GetNamespace() + "/test"
		// when
		err := checkUpgradeability(localDogu, remoteDogu, false)

		// then
		require.Error(t, err)
		assert.Equal(t, "upgrade-ability check failed: dogus must have the same name (ldap=test)", err.Error())
	})
	t.Run("should fail for different dogu names", func(t *testing.T) {
		localDogu := readTestDataLdapDogu(t)
		remoteDogu := readTestDataLdapDogu(t)
		remoteDogu.Name = "different-ns/ldap"
		// when
		err := checkUpgradeability(localDogu, remoteDogu, false)

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
		testDeployment := createReadyDeployment("ldap")
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)

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
		testDeployment := createDeployment("ldap", 1, 0) // trigger failure

		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(testDeployment.GroupVersionKind(), &appsv1.Deployment{})
		myClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = defaultNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)

		// when
		err := sut.doguHealthChecker.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.Error(t, err)
		assert.Equal(t, "dogu failed a health check: dogu ldap appears unhealthy (desired replicas: 1, ready: 0)", err.Error())
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
		redmineCr := readTestDataRedmineCr(t)

		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		redmineDogu := readTestDataRedmineDogu(t)
		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		nginxDogu := readTestDataDogu(t, nginxBytes)
		casDogu := readTestDataDogu(t, casBytes)
		postfixDogu := readTestDataDogu(t, postfixBytes)
		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
		doguRegistry.On("Get", "nginx").Return(nginxDogu, nil)
		doguRegistry.On("Get", "cas").Return(casDogu, nil)
		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)

		dependentDeployment := createDeployment("redmine", 1, 0)
		dependencyDeployment1 := createReadyDeployment("postgresql")
		dependencyDeployment2 := createReadyDeployment("nginx")
		dependencyDeployment3 := createReadyDeployment("cas")
		dependencyDeployment4 := createReadyDeployment("postfix")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4).
			Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)
		dependencyValidatorMock := &mocks.DependencyValidator{}
		dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		sut.dependencyValidator = dependencyValidatorMock

		// when
		err := sut.checkDependencyDogusHealthy(context.TODO(), redmineCr, redmineDogu)

		// then
		require.NoError(t, err)
		cesRegistry.AssertExpectations(t)
		doguRegistry.AssertExpectations(t)
		dependencyValidatorMock.AssertExpectations(t)
	})
	t.Run("should fail when at least one dependency dogus is unhealthy", func(t *testing.T) {
		redmineCr := readTestDataRedmineCr(t)

		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		redmineDogu := readTestDataRedmineDogu(t)
		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		nginxDogu := readTestDataDogu(t, nginxBytes)
		casDogu := readTestDataDogu(t, casBytes)
		postfixDogu := readTestDataDogu(t, postfixBytes)
		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
		doguRegistry.On("Get", "nginx").Return(nginxDogu, nil)
		doguRegistry.On("Get", "cas").Return(casDogu, nil)
		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)

		dependentDeployment := createDeployment("redmine", 1, 0)
		dependencyDeployment1 := createReadyDeployment("postgresql")
		dependencyDeployment2 := createReadyDeployment("nginx")
		dependencyDeployment3 := createReadyDeployment("cas")
		dependencyDeployment4 := createDeployment("postfix", 1, 0) // boom

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4).
			Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)
		dependencyValidatorMock := &mocks.DependencyValidator{}
		dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		sut.dependencyValidator = dependencyValidatorMock

		// when
		err := sut.checkDependencyDogusHealthy(context.TODO(), redmineCr, redmineDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dogu failed a health check: dogu postfix appears unhealthy (desired replicas: 1, ready: 0)")
		cesRegistry.AssertExpectations(t)
		doguRegistry.AssertExpectations(t)
		dependencyValidatorMock.AssertExpectations(t)
	})
}

func Test_doguUpgradeManager_getDogusForResource(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should return installed dogu and remote upgrade", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineDogu := readTestDataRedmineDogu(t)
		redmineDoguUpgrade := readTestDataRedmineDogu(t)
		redmineDoguUpgrade.Version = upgradeVersion

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.On("Get", "redmine").Return(redmineDogu, nil)
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		remoteRegistryMock := &cesremotemocks.Registry{}
		remoteRegistryMock.On("GetVersion", "official/redmine", upgradeVersion).Return(redmineDoguUpgrade, nil)

		sut, err := NewDoguUpgradeManager(nil, operatorConfig, cesRegistry, nil)
		sut.doguRemoteRegistry = remoteRegistryMock

		// when
		localDogu, remoteDogu, err := sut.getDogusForResource(redmineCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, redmineDogu, localDogu)
		assert.Equal(t, redmineDoguUpgrade, remoteDogu)
		remoteRegistryMock.AssertExpectations(t)
	})
}

func Test_doguUpgradeManager_namespaceChange(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should return true when the namespace should be changed", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineDogu := readTestDataRedmineDogu(t)
		redmineDoguUpgrade := readTestDataRedmineDogu(t)
		redmineDoguUpgrade.Version = upgradeVersion

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.On("Get", "redmine").Return(redmineDogu, nil)
		cesRegistry := &cesmocks.Registry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		remoteRegistryMock := &cesremotemocks.Registry{}
		remoteRegistryMock.On("GetVersion", "official/redmine", upgradeVersion).Return(redmineDoguUpgrade, nil)

		sut, err := NewDoguUpgradeManager(nil, operatorConfig, cesRegistry, nil)
		sut.doguRemoteRegistry = remoteRegistryMock

		// when

		// then
		require.NoError(t, err)
	})
}

func Test_doguUpgradeManager_Upgrade(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace
	ctx := context.TODO()

	t.Run("should succeed when also the namespace should be changed", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDogu := readTestDataRedmineDogu(t)
		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		casDogu := readTestDataDogu(t, casBytes)
		nginxDogu := readTestDataDogu(t, nginxBytes)
		postfixDogu := readTestDataDogu(t, postfixBytes)

		redmineDoguUpgrade := readTestDataRedmineDogu(t)
		redmineDoguUpgrade.Version = upgradeVersion

		doguRegistry := new(cesmocks.DoguRegistry)
		doguRegistry.On("Get", "redmine").Return(redmineDogu, nil)
		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
		doguRegistry.On("Get", "cas").Return(casDogu, nil)
		doguRegistry.On("Get", "nginx").Return(nginxDogu, nil)
		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)

		cesRegistry := new(cesmocks.Registry)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		recorderMock := new(mocks.EventRecorder)
		recorderMock.On("Event", mock.Anything, corev1.EventTypeNormal, UpgradeEventReason, "Checking premises...")
		recorderMock.On("Event", mock.Anything, corev1.EventTypeNormal, UpgradeEventReason, "Checking upgradeability...")
		remoteRegistryMock := &cesremotemocks.Registry{}
		remoteRegistryMock.On("GetVersion", "official/redmine", upgradeVersion).Return(redmineDoguUpgrade, nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx := createReadyDeployment("nginx")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(deplRedmine, deplPostgres, deplCas, deplNginx, deplPostfix).
			Build()

		sut, _ := NewDoguUpgradeManager(clientMock, operatorConfig, cesRegistry, recorderMock)
		sut.doguRemoteRegistry = remoteRegistryMock

		// when
		err := sut.Upgrade(ctx, redmineCr)

		// then
		require.NoError(t, err)
		doguRegistry.AssertExpectations(t)
		cesRegistry.AssertExpectations(t)
		remoteRegistryMock.AssertExpectations(t)
		recorderMock.AssertExpectations(t)
	})
	t.Run("should fail and record error event", func(t *testing.T) {
		// given
		redmineCr := readTestDataRedmineCr(t)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDogu := readTestDataRedmineDogu(t)
		redmineDoguUpgrade := readTestDataRedmineDogu(t)
		redmineDoguUpgrade.Version = upgradeVersion

		doguRegistry := new(cesmocks.DoguRegistry)
		doguRegistry.On("Get", "redmine").Return(redmineDogu, nil)
		doguRegistry.On("Get", "postgresql").Return(nil, registryKeyNotFoundTestErr)
		doguRegistry.On("Get", "cas").Return(nil, registryKeyNotFoundTestErr)
		doguRegistry.On("Get", "postfix").Return(nil, registryKeyNotFoundTestErr)

		cesRegistry := new(cesmocks.Registry)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		recorderMock := new(mocks.EventRecorder)
		recorderMock.On("Event", mock.Anything, corev1.EventTypeNormal, UpgradeEventReason, "Checking premises...")
		recorderMock.On("Eventf", mock.Anything, corev1.EventTypeWarning, ErrorOnFailedPremisesUpgradeEventReason, "Checking premises failed: %s", mock.Anything)
		remoteRegistryMock := &cesremotemocks.Registry{}
		remoteRegistryMock.On("GetVersion", "official/redmine", upgradeVersion).Return(redmineDoguUpgrade, nil)

		deplRedmine := createReadyDeployment("redmine")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(deplRedmine).
			Build()

		sut, _ := NewDoguUpgradeManager(clientMock, operatorConfig, cesRegistry, recorderMock)
		sut.doguRemoteRegistry = remoteRegistryMock

		// when
		err := sut.Upgrade(ctx, redmineCr)

		// then
		require.Error(t, err)
		doguRegistry.AssertExpectations(t)
		cesRegistry.AssertExpectations(t)
		remoteRegistryMock.AssertExpectations(t)
		recorderMock.AssertExpectations(t)
	})
}
