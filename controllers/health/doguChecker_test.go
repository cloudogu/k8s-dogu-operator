package health

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/coreos/etcd/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "test-namespace"

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

var registryKeyNotFoundTestErr = &client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"}

func Test_doguChecker_checkDoguHealth(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig
	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		testDeployment := createDeployment("ldap", 1, 1)
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataPostgresqlCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.NoError(t, err)
		doguRegistry.AssertExpectations(t)
	})
	t.Run("should fail because of unready replicas", func(t *testing.T) {
		doguRegistry := &cesmocks.DoguRegistry{}
		testDeployment := createDeployment("ldap", 1, 0)
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataPostgresqlCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dogu ldap appears unhealthy (desired replicas: 1, ready: 0)")
		doguRegistry.AssertExpectations(t)
	})
}

func Test_doguChecker_checkDependencyDogusHealthy(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed when all dogu dependencies are in a healthy state", func(t *testing.T) {
		/*
			redmine
			+-m-> ☑️postgresql
			+-m-> ☑️mandatory1
			+-o-> ☑️optional1
				  +-m-> ☑️mandatory1
				  +-o-> ☑️optional2
						+-m-> ☑️mandatory2
		*/

		doguRegistry := &cesmocks.DoguRegistry{}

		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
		doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
		doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
		doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
		doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
		doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

		redmineDogu := readTestDataDogu(t, redmineBytes)
		dependentDeployment := createDeployment("redmine", 1, 0)
		dependencyDeployment1 := createDeployment("postgresql", 1, 1)
		dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
		dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
		dependencyDeployment4 := createDeployment("optional1", 1, 1)
		dependencyDeployment5 := createDeployment("optional2", 1, 1)

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
			Build()

		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

		// then
		require.NoError(t, err)
		doguRegistry.AssertExpectations(t)
	})
	t.Run("should fail because of multiple reasons", func(t *testing.T) {
		/*
			redmine
			+-m-> ❌️postgresql
			+-m-> ☑️mandatory1
			+-o-> ❌️optional1
				  +-m-> ☑️mandatory1
				  +-o-> ☑️optional2
						+-m-> ❌️mandatory2
		*/

		doguRegistry := &cesmocks.DoguRegistry{}

		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		doguRegistry.On("Get", "postgresql").Return(nil, registryKeyNotFoundTestErr)
		doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
		doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
		doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
		doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
		doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

		redmineDogu := readTestDataDogu(t, redmineBytes)
		dependentDeployment := createDeployment("redmine", 1, 0)
		// dependencyDeployment1 postgresql was not even asked because of missing registry config
		dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
		// dependencyDeployment3 deployment mandatory2 is missing
		dependencyDeployment4 := createDeployment("optional1", 1, 0) // is not ready
		dependencyDeployment5 := createDeployment("optional2", 1, 1)

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(dependentDeployment, dependencyDeployment2, dependencyDeployment4, dependencyDeployment5).
			Build()

		sut := NewDoguChecker(myClient, doguRegistry)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "3 errors occurred")
		assert.Contains(t, err.Error(), "error getting registry key for postgresql")
		assert.Contains(t, err.Error(), "dogu optional1 appears unhealthy")
		assert.Contains(t, err.Error(), `dogu mandatory2 health check failed: deployments.apps "mandatory2" not found`)
		doguRegistry.AssertExpectations(t)
	})

	t.Run("on direct dependencies", func(t *testing.T) {
		t.Run("which are mandatory", func(t *testing.T) {
			t.Run("should fail when at least one mandatory dependency dogu is not installed", func(t *testing.T) {
				/*
					redmine
					+-m-> ❌️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/

				doguRegistry := &cesmocks.DoguRegistry{}

				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				doguRegistry.On("Get", "postgresql").Return(nil, registryKeyNotFoundTestErr)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				// dependencyDeployment1 is not even existing
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment4 := createDeployment("optional1", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "error getting registry key for postgresql")
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but deployment does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ❌️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				// dependencyDeployment1 does not exist
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment4 := createDeployment("optional1", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu postgresql health check failed")
				assert.Contains(t, err.Error(), `deployments.apps "postgresql" not found`)
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but deployment is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ❌️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 0) // boom
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment4 := createDeployment("optional1", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu postgresql appears unhealthy (desired replicas: 1, ready: 0)")
				doguRegistry.AssertExpectations(t)
			})
		})
		t.Run("which are optional", func(t *testing.T) {
			t.Run("should fail when at least one optional dependency dogu is installed but deployment is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ❌️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 0)
				dependencyDeployment4 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu optional1 appears unhealthy (desired replicas: 1, ready: 0)")
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should succeed when at least one optional dependency dogu is not installed", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ~optional1~
						  +-m-> ~mandatory1~
						  +-o-> ~optional2~
								+-m-> ~mandatory2~
				*/

				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one optional dependency dogu is installed but deployment does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ❌️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				// dependencyDeployment1 does not exist
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment4 := createDeployment("optional1", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu postgresql health check failed")
				assert.Contains(t, err.Error(), `deployments.apps "postgresql" not found`)
				doguRegistry.AssertExpectations(t)
			})
		})
	})
	t.Run("on indirect dependencies", func(t *testing.T) {
		t.Run("which are mandatory", func(t *testing.T) {
			t.Run("should fail when at least one mandatory dependency dogu is not installed", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ❌️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}
				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)

				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("mandatory2", 1, 1)
				dependencyDeployment4 := createDeployment("optional1", 1, 1)
				dependencyDeployment5 := createDeployment("optional2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "error getting registry key for mandatory2")
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but deployment does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ❌️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)
				dependencyDeployment4 := createDeployment("optional2", 1, 1)
				// dependencyDeployment5 does not exists

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu mandatory2 health check failed")
				assert.Contains(t, err.Error(), `deployments.apps "mandatory2" not found`)
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but deployment is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ❌️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)
				dependencyDeployment4 := createDeployment("optional2", 1, 1)
				dependencyDeployment5 := createDeployment("mandatory2", 1, 0) // boom

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu mandatory2 appears unhealthy (desired replicas: 1, ready: 0)")
				doguRegistry.AssertExpectations(t)
			})
		})
		t.Run("which are optional", func(t *testing.T) {
			t.Run("should fail when at least one optional dependency dogu is installed but deployment is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ❌️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)
				dependencyDeployment4 := createDeployment("optional2", 1, 0)
				dependencyDeployment5 := createDeployment("mandatory2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment4, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu optional2 appears unhealthy (desired replicas: 1, ready: 0)")
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should fail when at least one optional dependency dogu is installed but deployment does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ❌️optional2
								+-m-> ☑️mandatory2
				*/
				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(optional2Dogu, nil)
				doguRegistry.On("Get", "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)
				// dependencyDeployment4 is missing
				dependencyDeployment5 := createDeployment("mandatory2", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3, dependencyDeployment5).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Contains(t, err.Error(), "1 error occurred")
				assert.Contains(t, err.Error(), "dogu optional2 health check failed")
				assert.Contains(t, err.Error(), `deployments.apps "optional2" not found`)
				doguRegistry.AssertExpectations(t)
			})
			t.Run("should succeed when at least one optional dependency dogu is not installed", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑ optional1
						  +-m-> ☑ mandatory1
						  +-o-> ~optional2~
								+-m-> ~mandatory2~
				*/

				doguRegistry := &cesmocks.DoguRegistry{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				doguRegistry.On("Get", "postgresql").Return(postgresqlDogu, nil)
				doguRegistry.On("Get", "mandatory1").Return(mandatory1Dogu, nil)
				doguRegistry.On("Get", "optional1").Return(optional1Dogu, nil)
				doguRegistry.On("Get", "optional2").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3).
					Build()

				sut := NewDoguChecker(myClient, doguRegistry)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
				doguRegistry.AssertExpectations(t)
			})

		})
	})
}

func createDeployment(doguName string, replicas, replicasReady int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: replicas, ReadyReplicas: replicasReady},
	}
}

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
}
