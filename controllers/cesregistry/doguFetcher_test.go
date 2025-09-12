package cesregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"

	"github.com/cloudogu/cesapp-lib/core"
)

var testCtx = context.Background()

func Test_localDoguFetcher_FetchInstalled(t *testing.T) {
	t.Run("should succeed and return installed dogu", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		coreDoguVersion, lerr := dogu.GetVersion()
		require.NoError(t, lerr)
		simpleDoguName := cescommons.SimpleName(dogu.GetSimpleName())
		doguVersion := cescommons.SimpleNameVersion{
			Name:    simpleDoguName,
			Version: coreDoguVersion,
		}

		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleDoguName).Return(doguVersion, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Get(testCtx, doguVersion).Return(dogu, nil)

		sut := NewLocalDoguFetcher(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		installedDogu, err := sut.FetchInstalled(testCtx, doguCr.GetSimpleDoguName())

		// then
		require.NoError(t, err)
		assert.Equal(t, dogu, installedDogu)
	})
	t.Run("should fail to get dogu from local registry", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		coreDoguVersion, lerr := dogu.GetVersion()
		require.NoError(t, lerr)
		simpleDoguName := cescommons.SimpleName(dogu.GetSimpleName())
		doguVersion := cescommons.SimpleNameVersion{
			Name:    simpleDoguName,
			Version: coreDoguVersion,
		}

		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleDoguName).Return(doguVersion, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Get(testCtx, doguVersion).Return(dogu, assert.AnError)

		sut := NewLocalDoguFetcher(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		_, err := sut.FetchInstalled(testCtx, doguCr.GetSimpleDoguName())

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get local dogu descriptor for redmine")
	})
	t.Run("should return a dogu that misses a no-substitute dependency", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		registratorDep := core.Dependency{
			Name:    "registrator",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		dogu.Dependencies = append(dogu.Dependencies, registratorDep)
		require.Contains(t, dogu.Dependencies, registratorDep)
		coreDoguVersion, lerr := dogu.GetVersion()
		require.NoError(t, lerr)
		simpleDoguName := cescommons.SimpleName(dogu.GetSimpleName())
		doguVersion := cescommons.SimpleNameVersion{
			Name:    simpleDoguName,
			Version: coreDoguVersion,
		}

		mockDoguVersionRegistry := newMockDoguVersionRegistry(t)
		mockDoguVersionRegistry.EXPECT().GetCurrent(testCtx, simpleDoguName).Return(doguVersion, nil)
		mockLocalDoguDescriptorRepository := newMockLocalDoguDescriptorRepository(t)
		mockLocalDoguDescriptorRepository.EXPECT().Get(testCtx, doguVersion).Return(dogu, nil)

		sut := NewLocalDoguFetcher(mockDoguVersionRegistry, mockLocalDoguDescriptorRepository)

		// when
		installedDogu, err := sut.FetchInstalled(testCtx, doguCr.GetSimpleDoguName())

		// then
		require.NoError(t, err)
		assert.NotContains(t, installedDogu.Dependencies, core.Dependency{Name: "registrator", Type: core.DependencyTypeDogu})
	})
}

func Test_resourceDoguFetcher_FetchFromResource(t *testing.T) {
	t.Run("should fail to retrieve dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		remoteDoguRepo := newMockRemoteDoguDescriptorRepository(t)
		client := NewMockK8sClient(t)
		client.EXPECT().Get(testCtx, doguCr.GetDevelopmentDoguMapKey(), mock.AnythingOfType("*v1.ConfigMap")).Return(assert.AnError)
		sut := NewResourceDoguFetcher(client, remoteDoguRepo)

		// when
		_, _, err := sut.FetchWithResource(testCtx, doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get development dogu map: failed to get development dogu map for dogu redmine")
		mock.AssertExpectationsForObjects(t, client, remoteDoguRepo)
	})
	t.Run("should fail on missing dogu development map and missing remote dogu", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		resourceNotFoundErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, doguCr.GetDevelopmentDoguMapKey().Name)

		client := NewMockK8sClient(t)
		client.EXPECT().Get(testCtx, doguCr.GetDevelopmentDoguMapKey(), mock.AnythingOfType("*v1.ConfigMap")).Return(resourceNotFoundErr)

		remoteDoguRepo := newMockRemoteDoguDescriptorRepository(t)
		remoteDoguRepo.EXPECT().Get(testCtx, cescommons.QualifiedVersion{Name: cescommons.QualifiedName{SimpleName: "redmine", Namespace: "official"}, Version: core.Version{Raw: "4.2.3-10", Major: 4, Minor: 2, Patch: 3, Nano: 0, Extra: 10}}).Return(&core.Dogu{}, assert.AnError)

		sut := NewResourceDoguFetcher(client, remoteDoguRepo)

		// when
		_, _, err := sut.FetchWithResource(testCtx, doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu from remote or cache")
		mock.AssertExpectationsForObjects(t, client, remoteDoguRepo)
	})
	t.Run("should fetch dogu from dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		expectedDevelopmentDoguMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(expectedDevelopmentDoguMap.ToConfigMap()).Build()
		sut := NewResourceDoguFetcher(client, nil)

		// when
		fetchedDogu, DevelopmentDoguMap, err := sut.FetchWithResource(testCtx, doguCr)

		// then
		require.NoError(t, err)
		expectedDogu := readTestDataDogu(t, redmineBytes)

		for idx, dep := range expectedDogu.Dependencies {
			if dep.Name == "nginx" {
				expectedDogu.Dependencies = append(expectedDogu.Dependencies[:idx], expectedDogu.Dependencies[idx+1:]...)
			}
		}
		// save the dependencies for later
		expectedDependencies := expectedDogu.Dependencies
		actualDependencies := fetchedDogu.Dependencies
		expectedOptionalDependencies := expectedDogu.OptionalDependencies
		actualOptionalDependencies := fetchedDogu.OptionalDependencies

		// reset dependencies otherwise the dogu equality test sucks like awfully bad for the tiniest change
		expectedDogu.Dependencies = nil
		fetchedDogu.Dependencies = nil
		expectedDogu.OptionalDependencies = nil
		fetchedDogu.OptionalDependencies = nil
		assert.Equal(t, expectedDogu, fetchedDogu)

		assert.ElementsMatch(t, expectedDependencies, actualDependencies)
		assert.ElementsMatch(t, expectedOptionalDependencies, actualOptionalDependencies)
		assert.Equal(t, expectedDevelopmentDoguMap.Name, DevelopmentDoguMap.Name)
	})
	t.Run("should fetch dogu from remote registry", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		testDogu := readTestDataDogu(t, redmineBytes)

		remoteDoguRepo := newMockRemoteDoguDescriptorRepository(t)
		remoteDoguRepo.EXPECT().Get(testCtx, cescommons.QualifiedVersion{Name: cescommons.QualifiedName{SimpleName: "redmine", Namespace: "official"}, Version: core.Version{Raw: "4.2.3-10", Major: 4, Minor: 2, Patch: 3, Nano: 0, Extra: 10}}).Return(testDogu, nil)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		sut := NewResourceDoguFetcher(client, remoteDoguRepo)

		// when
		fetchedDogu, cleanup, err := sut.FetchWithResource(testCtx, doguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, testDogu, fetchedDogu)
		assert.Nil(t, cleanup)
		mock.AssertExpectationsForObjects(t, remoteDoguRepo)
	})
	t.Run("should return a dogu that misses a no-substitute dependency", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		testDogu := readTestDataDogu(t, redmineBytes)
		registratorDep := core.Dependency{
			Name:    "registrator",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		testDogu.Dependencies = append(testDogu.Dependencies, registratorDep)
		require.Contains(t, testDogu.Dependencies, registratorDep)

		remoteDoguRepo := newMockRemoteDoguDescriptorRepository(t)
		remoteDoguRepo.EXPECT().Get(testCtx, cescommons.QualifiedVersion{Name: cescommons.QualifiedName{SimpleName: "redmine", Namespace: "official"}, Version: core.Version{Raw: "4.2.3-10", Major: 4, Minor: 2, Patch: 3, Nano: 0, Extra: 10}}).Return(&core.Dogu{}, nil)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		sut := NewResourceDoguFetcher(client, remoteDoguRepo)

		// when
		fetchedDogu, _, err := sut.FetchWithResource(testCtx, doguCr)

		// then
		require.NoError(t, err)
		require.NotContains(t, fetchedDogu.Dependencies, core.Dependency{Name: "registrator", Type: core.DependencyTypeDogu})
		mock.AssertExpectationsForObjects(t, remoteDoguRepo)
	})
}

func Test_resourceDoguFetcher_getFromDevelopmentDoguMap(t *testing.T) {
	t.Run("fail as config map contains invalid json", func(t *testing.T) {
		// given
		sut := NewResourceDoguFetcher(nil, nil)
		redmineDevelopmentDoguMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)
		redmineDevelopmentDoguMap.Data["dogu.json"] = "invalid dogu json"

		// when
		_, err := sut.getFromDevelopmentDoguMap(redmineDevelopmentDoguMap)

		// given
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal custom dogu descriptor")
	})
}

func Test_doguFetcher_getDoguFromRemoteRegistry(t *testing.T) {
	t.Run("fail when remote registry returns error", func(t *testing.T) {
		// given
		doguVersion := readTestDataRedmineQualifiedDoguVersion(t)

		remoteDoguRepo := newMockRemoteDoguDescriptorRepository(t)
		remoteDoguRepo.EXPECT().Get(context.TODO(), *doguVersion).Return(&core.Dogu{}, assert.AnError)

		sut := NewResourceDoguFetcher(nil, remoteDoguRepo)

		// when
		_, err := sut.getDoguFromRemoteRegistry(context.TODO(), *doguVersion)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu from remote dogu registry")
		mock.AssertExpectationsForObjects(t, remoteDoguRepo)
	})
}
