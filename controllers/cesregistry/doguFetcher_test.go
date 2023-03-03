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

	"github.com/cloudogu/cesapp-lib/core"
	mocks2 "github.com/cloudogu/cesapp-lib/registry/mocks"
	mocks3 "github.com/cloudogu/cesapp-lib/remote/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

var ctx = context.Background()

func Test_localDoguFetcher_FetchInstalled(t *testing.T) {
	t.Run("should succeed and return installed dogu", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)

		localRegDoguContextMock := new(mocks2.DoguRegistry)
		localRegDoguContextMock.On("Get", "redmine").Return(dogu, nil)

		sut := NewLocalDoguFetcher(localRegDoguContextMock)

		// when
		installedDogu, err := sut.FetchInstalled(doguCr.Name)

		// then
		require.NoError(t, err)
		assert.Equal(t, dogu, installedDogu)
		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
	})
	t.Run("should fail to get dogu from local registry", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)

		localRegDoguContextMock := new(mocks2.DoguRegistry)
		localRegDoguContextMock.On("Get", "redmine").Return(nil, assert.AnError)

		sut := NewLocalDoguFetcher(localRegDoguContextMock)

		// when
		_, err := sut.FetchInstalled(doguCr.Name)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get local dogu descriptor for redmine")
		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
	})
	t.Run("should return a dogu with K8s-CES compatible substitutes for an nginx dependency", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		expectedIncompatibleDepNginx := core.Dependency{
			Name:    "nginx",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		require.Contains(t, dogu.Dependencies, expectedIncompatibleDepNginx)

		localRegDoguContextMock := new(mocks2.DoguRegistry)
		localRegDoguContextMock.On("Get", "redmine").Return(dogu, nil)

		sut := NewLocalDoguFetcher(localRegDoguContextMock)

		// when
		installedDogu, err := sut.FetchInstalled(doguCr.Name)

		// then
		require.NoError(t, err)
		expectedNginxPatch1 := core.Dependency{
			Name:    "nginx-ingress",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		expectedNginxPatch2 := core.Dependency{
			Name:    "nginx-static",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}

		unexpectedAfterPatch := expectedIncompatibleDepNginx
		assert.Contains(t, installedDogu.Dependencies, expectedNginxPatch1)
		assert.Contains(t, installedDogu.Dependencies, expectedNginxPatch2)
		assert.NotContains(t, installedDogu.Dependencies, unexpectedAfterPatch)
		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
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

		localRegDoguContextMock := new(mocks2.DoguRegistry)
		localRegDoguContextMock.On("Get", "redmine").Return(dogu, nil)
		sut := NewLocalDoguFetcher(localRegDoguContextMock)

		// when
		installedDogu, err := sut.FetchInstalled(doguCr.Name)

		// then
		require.NoError(t, err)
		assert.NotContains(t, installedDogu.Dependencies, core.Dependency{Name: "registrator", Type: core.DependencyTypeDogu})
		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
	})
}

func Test_resourceDoguFetcher_FetchFromResource(t *testing.T) {
	t.Run("should fail to retrieve dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)

		remoteDoguRegistry := new(mocks3.Registry)
		client := extMocks.NewK8sClient(t)
		client.EXPECT().Get(ctx, doguCr.GetDevelopmentDoguMapKey(), mock.AnythingOfType("*v1.ConfigMap")).Return(assert.AnError)
		sut := NewResourceDoguFetcher(client, remoteDoguRegistry)

		// when
		_, _, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get development dogu map: failed to get development dogu map for dogu redmine")
		mock.AssertExpectationsForObjects(t, client, remoteDoguRegistry)
	})
	t.Run("should fail on missing dogu development map and missing remote dogu", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		resourceNotFoundErr := errors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, doguCr.GetDevelopmentDoguMapKey().Name)

		client := extMocks.NewK8sClient(t)
		client.EXPECT().Get(ctx, doguCr.GetDevelopmentDoguMapKey(), mock.AnythingOfType("*v1.ConfigMap")).Return(resourceNotFoundErr)

		remoteDoguRegistry := new(mocks3.Registry)
		remoteDoguRegistry.On("GetVersion", doguCr.Spec.Name, doguCr.Spec.Version).Return(nil, assert.AnError)
		sut := NewResourceDoguFetcher(client, remoteDoguRegistry)

		// when
		_, _, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu from remote or cache")
		mock.AssertExpectationsForObjects(t, client, remoteDoguRegistry)
	})
	t.Run("should fetch dogu from dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		expectedDevelopmentDoguMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(expectedDevelopmentDoguMap.ToConfigMap()).Build()
		sut := NewResourceDoguFetcher(client, nil)

		// when
		fetchedDogu, DevelopmentDoguMap, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.NoError(t, err)
		expectedDogu := readTestDataDogu(t, redmineBytes)
		expectedNginxPatch1 := core.Dependency{
			Name:    "nginx-ingress",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		expectedNginxPatch2 := core.Dependency{
			Name:    "nginx-static",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		for idx, dep := range expectedDogu.Dependencies {
			if dep.Name == "nginx" {
				expectedDogu.Dependencies = append(expectedDogu.Dependencies[:idx], expectedDogu.Dependencies[idx+1:]...)
				expectedDogu.Dependencies = append(expectedDogu.Dependencies, expectedNginxPatch1)
				expectedDogu.Dependencies = append(expectedDogu.Dependencies, expectedNginxPatch2)
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
		dogu := readTestDataDogu(t, redmineBytes)

		remoteDoguRegistry := new(mocks3.Registry)
		remoteDoguRegistry.On("GetVersion", doguCr.Spec.Name, doguCr.Spec.Version).Return(dogu, nil)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		sut := NewResourceDoguFetcher(client, remoteDoguRegistry)

		// when
		fetchedDogu, cleanup, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, dogu, fetchedDogu)
		assert.Nil(t, cleanup)
		mock.AssertExpectationsForObjects(t, remoteDoguRegistry)
	})
	t.Run("should return a dogu with K8s-CES compatible substitutes for an nginx dependency", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		expectedIncompatibleDepNginx := core.Dependency{
			Name:    "nginx",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		require.Contains(t, dogu.Dependencies, expectedIncompatibleDepNginx)

		localRegDoguContextMock := new(mocks2.DoguRegistry)

		redmineDevelopmentDoguMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(redmineDevelopmentDoguMap.ToConfigMap()).Build()
		sut := NewResourceDoguFetcher(client, nil)

		// when
		fetchedDogu, _, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.NoError(t, err)
		expectedNginxPatch1 := core.Dependency{
			Name:    "nginx-ingress",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}
		expectedNginxPatch2 := core.Dependency{
			Name:    "nginx-static",
			Version: "",
			Type:    core.DependencyTypeDogu,
		}

		unexpectedAfterPatch := expectedIncompatibleDepNginx
		assert.Contains(t, fetchedDogu.Dependencies, expectedNginxPatch1)
		assert.Contains(t, fetchedDogu.Dependencies, expectedNginxPatch2)
		assert.NotContains(t, fetchedDogu.Dependencies, unexpectedAfterPatch)

		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
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

		remoteRegMock := new(mocks3.Registry)
		remoteRegMock.On("GetVersion", "official/redmine", "4.2.3-10").Return(dogu, nil)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		sut := NewResourceDoguFetcher(client, remoteRegMock)

		// when
		fetchedDogu, _, err := sut.FetchWithResource(ctx, doguCr)

		// then
		require.NoError(t, err)
		require.NotContains(t, fetchedDogu.Dependencies, core.Dependency{Name: "registrator", Type: core.DependencyTypeDogu})
		mock.AssertExpectationsForObjects(t, remoteRegMock)
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
		doguCr := readTestDataRedmineCr(t)

		remoteDoguRegistry := new(mocks3.Registry)
		remoteDoguRegistry.On("GetVersion", doguCr.Spec.Name, doguCr.Spec.Version).Return(nil, assert.AnError)

		sut := NewResourceDoguFetcher(nil, remoteDoguRegistry)

		// when
		_, err := sut.getDoguFromRemoteRegistry(doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu from remote dogu registry")
		mock.AssertExpectationsForObjects(t, remoteDoguRegistry)
	})
}
