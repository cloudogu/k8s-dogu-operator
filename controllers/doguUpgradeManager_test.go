package controllers

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"

	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultNamespace = ""

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

var textCtx = context.TODO()

var registryKeyNotFoundTestErr = client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"}

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

// func Test_doguUpgradeManager_checkDependencyDogusHealthy(t *testing.T) {
// 	// override default controller method to retrieve a kube config
// 	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
// 	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
// 	ctrl.GetConfigOrDie = createTestRestConfig
//
// 	operatorConfig := &config.OperatorConfig{}
// 	operatorConfig.Namespace = testNamespace
//
// 	t.Run("should succeed when all dogu dependencies are in a healthy state", func(t *testing.T) {
// 		redmineCr := readTestDataRedmineCr(t)
//
// 		doguRegistry := &cesmocks.DoguRegistry{}
// 		cesRegistry := &cesmocks.Registry{}
// 		cesRegistry.On("DoguRegistry").Return(doguRegistry)
//
// 		redmineDogu := readTestDataRedmineDogu(t)
// 		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
// 		nginxDogu := readTestDataDogu(t, nginxBytes)
// 		casDogu := readTestDataDogu(t, casBytes)
// 		postfixDogu := readTestDataDogu(t, postfixBytes)
// 		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
// 		doguRegistry.On("Get", "nginx").Return(nginxDogu, nil)
// 		doguRegistry.On("Get", "cas").Return(casDogu, nil)
// 		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)
//
// 		dependentDeployment := createDeployment("redmine", 1, 0)
// 		dependencyDeployment1 := createReadyDeployment("postgresql")
// 		dependencyDeployment2 := createReadyDeployment("nginx")
// 		dependencyDeployment3 := createReadyDeployment("cas")
// 		dependencyDeployment4 := createReadyDeployment("postfix")
//
// 		myClient := fake.NewClientBuilder().
// 			WithScheme(getTestScheme()).
// 			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4).
// 			Build()
//
// 		ldapResource := readTestDataLdapCr(t)
// 		ldapResource.Namespace = testNamespace
// 		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)
// 		dependencyValidatorMock := &mocks.DependencyValidator{}
// 		dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
// 		sut.dependencyValidator = dependencyValidatorMock
//
// 		// when
// 		err := sut.checkDependencyDogusHealthy(context.TODO(), redmineCr, redmineDogu)
//
// 		// then
// 		require.NoError(t, err)
// 		cesRegistry.AssertExpectations(t)
// 		doguRegistry.AssertExpectations(t)
// 		dependencyValidatorMock.AssertExpectations(t)
// 	})
// 	t.Run("should fail when at least one dependency dogus is unhealthy", func(t *testing.T) {
// 		redmineCr := readTestDataRedmineCr(t)
//
// 		doguRegistry := &cesmocks.DoguRegistry{}
// 		cesRegistry := &cesmocks.Registry{}
// 		cesRegistry.On("DoguRegistry").Return(doguRegistry)
//
// 		redmineDogu := readTestDataRedmineDogu(t)
// 		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
// 		nginxDogu := readTestDataDogu(t, nginxBytes)
// 		casDogu := readTestDataDogu(t, casBytes)
// 		postfixDogu := readTestDataDogu(t, postfixBytes)
// 		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
// 		doguRegistry.On("Get", "nginx").Return(nginxDogu, nil)
// 		doguRegistry.On("Get", "cas").Return(casDogu, nil)
// 		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)
//
// 		dependentDeployment := createDeployment("redmine", 1, 0)
// 		dependencyDeployment1 := createReadyDeployment("postgresql")
// 		dependencyDeployment2 := createReadyDeployment("nginx")
// 		dependencyDeployment3 := createReadyDeployment("cas")
// 		dependencyDeployment4 := createDeployment("postfix", 1, 0) // boom
//
// 		myClient := fake.NewClientBuilder().
// 			WithScheme(getTestScheme()).
// 			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4).
// 			Build()
//
// 		ldapResource := readTestDataLdapCr(t)
// 		ldapResource.Namespace = testNamespace
// 		sut, _ := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)
// 		dependencyValidatorMock := &mocks.DependencyValidator{}
// 		dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
// 		sut.dependencyValidator = dependencyValidatorMock
//
// 		// when
// 		err := sut.checkDependencyDogusHealthy(context.TODO(), redmineCr, redmineDogu)
//
// 		// then
// 		require.Error(t, err)
// 		assert.Contains(t, err.Error(), "dogu failed a health check: dogu postfix appears unhealthy (desired replicas: 1, ready: 0)")
// 		cesRegistry.AssertExpectations(t)
// 		doguRegistry.AssertExpectations(t)
// 		dependencyValidatorMock.AssertExpectations(t)
// 	})
// }

func Test_doguUpgradeManager_Upgrade(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

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
		doguRegistry.On("Get", "nginx-ingress").Return(nginxDogu, nil) // rewritten from
		doguRegistry.On("Get", "nginx-static").Return(nginxDogu, nil)  // dogu fetcher
		doguRegistry.On("Get", "postfix").Return(postfixDogu, nil)

		cesRegistry := new(cesmocks.Registry)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		recorderMock := mocks.NewEventRecorder(t)
		recorderMock.On("Event", mock.Anything, corev1.EventTypeNormal, UpgradeEventReason, "Checking premises...")
		recorderMock.On("Eventf", mock.Anything, corev1.EventTypeNormal, UpgradeEventReason, "Executing upgrade from %s to %s...", mock.Anything)
		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", ctx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut, _ := NewDoguUpgradeManager(clientMock, operatorConfig, cesRegistry, recorderMock)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(ctx, redmineCr)

		// then
		require.NoError(t, err)
		doguRegistry.AssertExpectations(t)
		cesRegistry.AssertExpectations(t)
		// the other mocks assert their expectations during t.CleanUp()
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
		recorderMock := mocks.NewEventRecorder(t)
		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", textCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		deplRedmine := createReadyDeployment("redmine")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(deplRedmine).
			Build()

		sut, _ := NewDoguUpgradeManager(clientMock, operatorConfig, cesRegistry, recorderMock)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(ctx, redmineCr)

		// then
		require.Error(t, err)
		doguRegistry.AssertExpectations(t)
		cesRegistry.AssertExpectations(t)
		// the other mocks assert their expectations during t.CleanUp()
	})
}
