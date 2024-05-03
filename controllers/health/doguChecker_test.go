package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.etcd.io/etcd/client/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/cloudogu/cesapp-lib/core"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

const testNamespace = "test-namespace"

var registryKeyNotFoundTestErr = client.Error{Code: client.ErrorCodeKeyNotFound, Message: "Key not found"}
var testCtx = context.Background()

func Test_doguChecker_checkDoguHealth(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
	ctrl.GetConfig = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed", func(t *testing.T) {
		localFetcher := mocks.NewLocalDoguFetcher(t)

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace
		ldapResource.Status.Health = "available"

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, ldapResource.Name, metav1.GetOptions{}).Return(ldapResource, nil)
		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClientMock, localFetcher)

		// when
		err := sut.CheckByName(testCtx, ldapResource.GetObjectKey())

		// then
		require.NoError(t, err)
		localFetcher.AssertExpectations(t)
	})
	t.Run("should fail to get dogu cr", func(t *testing.T) {
		localFetcher := mocks.NewLocalDoguFetcher(t)

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, ldapResource.Name, metav1.GetOptions{}).Return(nil, assert.AnError)
		ecosystemClient := mocks.NewEcosystemInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClient, localFetcher)

		// when
		err := sut.CheckByName(testCtx, ldapResource.GetObjectKey())

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/ldap\"")
		localFetcher.AssertExpectations(t)
	})
	t.Run("should fail because of unready replicas", func(t *testing.T) {
		localFetcher := mocks.NewLocalDoguFetcher(t)

		ldapResource := readTestDataLdapCr(t)
		ldapResource.Namespace = testNamespace

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, ldapResource.Name, metav1.GetOptions{}).Return(ldapResource, nil)
		ecosystemClient := mocks.NewEcosystemInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClient, localFetcher)

		// when
		err := sut.CheckByName(testCtx, ldapResource.GetObjectKey())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu failed a health check: dogu \"ldap\" appears unhealthy")
		localFetcher.AssertExpectations(t)
	})
}

func Test_doguChecker_checkDependencyDogusHealthy(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfig = createTestRestConfig

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

		localFetcher := mocks.NewLocalDoguFetcher(t)

		postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

		redmineDogu := readTestDataDogu(t, redmineBytes)

		dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
		dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
		dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
		dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
		dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
		doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
		doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
		doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
		doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
		ecosystemClient := mocks.NewEcosystemInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

		// then
		require.NoError(t, err)
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

		localFetcher := mocks.NewLocalDoguFetcher(t)

		localFetcher.EXPECT().FetchInstalled(testCtx, "testDogu2").Once().Return(testDogu2, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "testDogu3").Once().Return(testDogu3, nil)

		dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "testDogu2"}, Status: doguv1.DoguStatus{Health: "available"}}
		dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "testDogu3"}, Status: doguv1.DoguStatus{Health: "available"}}

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "testDogu2", metav1.GetOptions{}).Return(dependencyResource2, nil)
		doguClientMock.EXPECT().Get(testCtx, "testDogu3", metav1.GetOptions{}).Return(dependencyResource3, nil)
		ecosystemClient := mocks.NewEcosystemInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(testCtx, testDogu, testNamespace)

		// then
		require.NoError(t, err)
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

		localFetcher := mocks.NewLocalDoguFetcher(t)

		mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
		optional1Dogu := readTestDataDogu(t, optional1Bytes)
		optional2Dogu := readTestDataDogu(t, optional2Bytes)
		mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, registryKeyNotFoundTestErr)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

		redmineDogu := readTestDataDogu(t, redmineBytes)

		// dependencyResource1 postgresql was not even asked because of missing registry config
		dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
		// dependencyResource3 mandatory2 is missing
		notFoundError := errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "mandatory2")
		dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "unavailable"}}
		dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

		doguClientMock := mocks.NewDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
		doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(nil, notFoundError)
		doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
		doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
		ecosystemClient := mocks.NewEcosystemInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := NewDoguChecker(ecosystemClient, localFetcher)

		// when
		err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

		// then
		require.Error(t, err)
		assert.Equal(t, 2, countMultiErrors(err))
		assert.ErrorContains(t, err, "error getting registry key for \"test-namespace/postgresql\"")                                          // the wrapping error
		assert.ErrorContains(t, err, "dogu \"optional1\" appears unhealthy")                                                                  // wrapped error 1
		assert.ErrorContains(t, err, `failed to get dogu resource "test-namespace/mandatory2": Dogu.k8s.cloudogu.com "mandatory2" not found`) // wrapped error 2
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

				localFetcher := mocks.NewLocalDoguFetcher(t)

				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, registryKeyNotFoundTestErr)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				// dependencyResource1 is not even existing
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "error getting registry key for \"test-namespace/postgresql\"")
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but dogu resource does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ❌️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				// dependencyResource1 does not exist
				notFoundError := errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "postgresql")
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(nil, notFoundError)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/postgresql\"")
				assert.ErrorContains(t, err, `Dogu.k8s.cloudogu.com "postgresql" not found`)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ❌️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "unavailable"}} // boom
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "dogu \"postgresql\" appears unhealthy")
			})
		})
		t.Run("which are optional", func(t *testing.T) {
			t.Run("should fail when at least one optional dependency dogu is installed but is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ❌️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "unavailable"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "dogu \"optional1\" appears unhealthy")
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

				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
			})
			t.Run("should fail when at least one optional dependency dogu is installed but dogu resource does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ❌️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				// dependencyResource1 does not exist
				notFoundError := errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "postgresql")
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(nil, notFoundError)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/postgresql\"")
				assert.ErrorContains(t, err, `Dogu.k8s.cloudogu.com "postgresql" not found`)
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
				localFetcher := mocks.NewLocalDoguFetcher(t)
				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)

				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "error getting registry key for \"test-namespace/mandatory2\"")
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but dogu resource does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ❌️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)

				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				// dependencyResource3 does not exists
				notFoundError := errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "mandatory2")
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(nil, notFoundError)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/mandatory2\"")
				assert.ErrorContains(t, err, `Dogu.k8s.cloudogu.com "mandatory2" not found`)
			})
			t.Run("should fail when at least one mandatory dependency dogu is installed but is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ☑️optional2
								+-m-> ❌️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "unavailable"}} // boom
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "dogu \"mandatory2\" appears unhealthy")
			})
		})
		t.Run("which are optional", func(t *testing.T) {
			t.Run("should fail when at least one optional dependency dogu is installed but is not ready", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ❌️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "unavailable"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "dogu \"optional2\" appears unhealthy")
			})
			t.Run("should fail when at least one optional dependency dogu is installed but dogu resource does not exist", func(t *testing.T) {
				/*
					redmine
					+-m-> ☑️postgresql
					+-m-> ☑️mandatory1
					+-o-> ☑️optional1
						  +-m-> ☑️mandatory1
						  +-o-> ❌️optional2
								+-m-> ☑️mandatory2
				*/
				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				optional2Dogu := readTestDataDogu(t, optional2Bytes)
				mandatory2Dogu := readTestDataDogu(t, mandatory2Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(optional2Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory2").Return(mandatory2Dogu, nil)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource3 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory2"}, Status: doguv1.DoguStatus{Health: "available"}}
				// dependencyResource4 is missing
				notFoundError := errors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "Dogu"}, "optional1")
				dependencyResource5 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional2"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory2", metav1.GetOptions{}).Return(dependencyResource3, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(nil, notFoundError)
				doguClientMock.EXPECT().Get(testCtx, "optional2", metav1.GetOptions{}).Return(dependencyResource5, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.Error(t, err)
				assert.Equal(t, 1, countMultiErrors(err))
				assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/optional1\"")
				assert.ErrorContains(t, err, `Dogu.k8s.cloudogu.com "optional1" not found`)
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

				localFetcher := mocks.NewLocalDoguFetcher(t)

				postgresqlDogu := readTestDataDogu(t, postgresqlBytes)
				mandatory1Dogu := readTestDataDogu(t, mandatory1Bytes)
				optional1Dogu := readTestDataDogu(t, optional1Bytes)
				localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "mandatory1").Return(mandatory1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional1").Return(optional1Dogu, nil)
				localFetcher.EXPECT().FetchInstalled(testCtx, "optional2").Return(nil, registryKeyNotFoundTestErr)

				redmineDogu := readTestDataDogu(t, redmineBytes)

				dependencyResource1 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "postgresql"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource2 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "mandatory1"}, Status: doguv1.DoguStatus{Health: "available"}}
				dependencyResource4 := &doguv1.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "optional1"}, Status: doguv1.DoguStatus{Health: "available"}}

				doguClientMock := mocks.NewDoguInterface(t)
				doguClientMock.EXPECT().Get(testCtx, "postgresql", metav1.GetOptions{}).Return(dependencyResource1, nil)
				doguClientMock.EXPECT().Get(testCtx, "mandatory1", metav1.GetOptions{}).Return(dependencyResource2, nil)
				doguClientMock.EXPECT().Get(testCtx, "optional1", metav1.GetOptions{}).Return(dependencyResource4, nil)
				ecosystemClient := mocks.NewEcosystemInterface(t)
				ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClientMock)

				sut := NewDoguChecker(ecosystemClient, localFetcher)

				// when
				err := sut.CheckDependenciesRecursive(testCtx, redmineDogu, testNamespace)

				// then
				require.NoError(t, err)
			})

		})
	})
}

func createTestRestConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

func countMultiErrors(err error) int {
	if err == nil {
		return 0
	}

	if unwrapped, ok := err.(interface{ Unwrap() []error }); ok {
		errs := unwrapped.Unwrap()
		return len(errs)
	}

	return 1
}
