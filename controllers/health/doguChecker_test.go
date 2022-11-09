package health

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/client/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
)

const testNamespace = "test-namespace"

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

var registryKeyNotFoundTestErr = client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"}

func Test_doguChecker_checkDoguHealth(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig
	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed", func(t *testing.T) {
		localFetcher := &mocks.LocalDoguFetcher{}
		testDeployment := createDeployment("ldap", 1, 1)
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, localFetcher)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.NoError(t, err)
		localFetcher.AssertExpectations(t)
	})
	t.Run("should fail because of unready replicas", func(t *testing.T) {
		localFetcher := &mocks.LocalDoguFetcher{}
		testDeployment := createDeployment("ldap", 1, 0)
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(testDeployment).Build()

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		sut := NewDoguChecker(myClient, localFetcher)

		// when
		err := sut.CheckWithResource(context.TODO(), ldapResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu ldap appears unhealthy (desired replicas: 1, ready: 0)")
		localFetcher.AssertExpectations(t)
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

		localFetcher := &mocks.LocalDoguFetcher{}

		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
		localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
		localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
		localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

		sut := NewDoguChecker(myClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

		// then
		require.NoError(t, err)
		localFetcher.AssertExpectations(t)
	})

	t.Run("should ignore client and package dependencies when checking health status of indirect dependencies", func(t *testing.T) {
		/*
			testDogu
			+-m-> ☑ client1 (client)
			+-m-> ☑ package1 (Package)
			+-m-> ☑ testDogu2 (Dogu)
				  +-o-> ☑ client2 (client)
				  +-o-> ☑ package2 (Package)
				  +-m-> ☑ testDogu3 (Dogu)
		*/

		testDogu := &core.Dogu{
			Name: "testDogu",
			Dependencies: []core.Dependency{
				{Type: core.DependencyTypeClient, Name: "client1"},
				{Type: core.DependencyTypePackage, Name: "package1"},
				{Type: core.DependencyTypeDogu, Name: "testDogu2"},
			},
		}
		testDogu2 := &core.Dogu{
			Name: "testDogu2",
			OptionalDependencies: []core.Dependency{
				{Type: core.DependencyTypeClient, Name: "client2"},
				{Type: core.DependencyTypePackage, Name: "package2"},
				{Type: core.DependencyTypeDogu, Name: "testDogu3"},
			},
		}
		testDogu3 := &core.Dogu{Name: "testDogu3"}

		localFetcher := &mocks.LocalDoguFetcher{}

		localFetcher.On("FetchInstalled", "testDogu2").Once().Return(testDogu2, nil)
		localFetcher.On("FetchInstalled", "testDogu3").Once().Return(testDogu3, nil)

		dependentDeployment := createDeployment("testDogu", 1, 0)
		dependencyDeployment1 := createDeployment("testDogu2", 1, 1)
		dependencyDeployment2 := createDeployment("testDogu3", 1, 1)

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2).
			Build()

		sut := NewDoguChecker(myClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), testDogu, testNamespace)

		// then
		require.NoError(t, err)
		localFetcher.AssertExpectations(t)
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

		localFetcher := &mocks.LocalDoguFetcher{}

		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		localFetcher.On("FetchInstalled", "postgresql").Return(nil, registryKeyNotFoundTestErr)
		localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
		localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
		localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

		sut := NewDoguChecker(myClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "3 errors occurred")
		assert.ErrorContains(t, err, "error getting registry key for postgresql")
		assert.ErrorContains(t, err, "dogu optional1 appears unhealthy")
		assert.ErrorContains(t, err, `dogu mandatory2 health check failed: deployments.apps "mandatory2" not found`)
		localFetcher.AssertExpectations(t)
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

				localFetcher := &mocks.LocalDoguFetcher{}

				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(nil, registryKeyNotFoundTestErr)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "error getting registry key for postgresql")
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu postgresql health check failed")
				assert.ErrorContains(t, err, `deployments.apps "postgresql" not found`)
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu postgresql appears unhealthy (desired replicas: 1, ready: 0)")
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu optional1 appears unhealthy (desired replicas: 1, ready: 0)")
				localFetcher.AssertExpectations(t)
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

				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2).
					Build()

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu postgresql health check failed")
				assert.ErrorContains(t, err, `deployments.apps "postgresql" not found`)
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}
				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)

				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(nil, registryKeyNotFoundTestErr)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "error getting registry key for mandatory2")
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu mandatory2 health check failed")
				assert.ErrorContains(t, err, `deployments.apps "mandatory2" not found`)
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu mandatory2 appears unhealthy (desired replicas: 1, ready: 0)")
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu optional2 appears unhealthy (desired replicas: 1, ready: 0)")
				localFetcher.AssertExpectations(t)
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
				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(optional2Dogu, nil)
				localFetcher.On("FetchInstalled", "mandatory2").Return(mandatory2Dogu, nil)

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

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.ErrorContains(t, err, "1 error occurred")
				assert.ErrorContains(t, err, "dogu optional2 health check failed")
				assert.ErrorContains(t, err, `deployments.apps "optional2" not found`)
				localFetcher.AssertExpectations(t)
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

				localFetcher := &mocks.LocalDoguFetcher{}

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				localFetcher.On("FetchInstalled", "postgresql").Return(postgresqlDogu, nil)
				localFetcher.On("FetchInstalled", "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional1").Return(optional1Dogu, nil)
				localFetcher.On("FetchInstalled", "optional2").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)
				dependentDeployment := createDeployment("redmine", 1, 0)
				dependencyDeployment1 := createDeployment("postgresql", 1, 1)
				dependencyDeployment2 := createDeployment("mandatory1", 1, 1)
				dependencyDeployment3 := createDeployment("optional1", 1, 1)

				myClient := fake.NewClientBuilder().
					WithScheme(getTestScheme()).
					WithObjects(dependentDeployment, dependencyDeployment1, dependencyDeployment2, dependencyDeployment3).
					Build()

				sut := NewDoguChecker(myClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(context.TODO(), redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
				localFetcher.AssertExpectations(t)
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
