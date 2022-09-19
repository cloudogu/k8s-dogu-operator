package upgrade

import (
	"context"
	"testing"

	mocks2 "github.com/cloudogu/cesapp-lib/registry/mocks"
	mocks3 "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-dogu-operator/api/v1/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_doguFetcher_FetchInstalled(t *testing.T) {
	t.Run("should succeed and return installed dogu", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)

		localRegDoguContextMock := new(mocks2.DoguRegistry)
		localRegDoguContextMock.On("Get", "redmine").Return(dogu, nil)

		client := &mocks.Client{}
		sut := NewDoguFetcher(client, localRegDoguContextMock, nil)

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

		client := &mocks.Client{}
		sut := NewDoguFetcher(client, localRegDoguContextMock, nil)

		// when
		_, err := sut.FetchInstalled(doguCr.Name)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to get local dogu descriptor for redmine")
		mock.AssertExpectationsForObjects(t, localRegDoguContextMock)
	})
}

func Test_doguFetcher_FetchFromResource(t *testing.T) {
	t.Run("should fail to retrieve dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)

		remoteDoguRegistry := new(mocks3.Registry)
		client := &mocks.Client{}
		client.On("Get", context.Background(), doguCr.GetDevelopmentDoguMapKey(), mock.AnythingOfType("*v1.ConfigMap")).Return(assert.AnError)
		sut := NewDoguFetcher(client, nil, remoteDoguRegistry)

		// when
		_, _, err := sut.FetchFromResource(context.Background(), doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to get development dogu map: failed to get development dogu map for dogu redmine")
		mock.AssertExpectationsForObjects(t, client, remoteDoguRegistry)
	})
	t.Run("should fetch dogu from dogu development map", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)
		expectedDoguDevelopmentMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(expectedDoguDevelopmentMap.ToConfigMap()).Build()
		sut := NewDoguFetcher(client, nil, nil)

		// when
		fetchedDogu, doguDevelopmentMap, err := sut.FetchFromResource(context.Background(), doguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, dogu, fetchedDogu)
		assert.Equal(t, expectedDoguDevelopmentMap.Name, doguDevelopmentMap.Name)
	})
	t.Run("should fetch dogu from remote registry", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)
		dogu := readTestDataDogu(t, redmineBytes)

		remoteDoguRegistry := new(mocks3.Registry)
		remoteDoguRegistry.On("GetVersion", doguCr.Spec.Name, doguCr.Spec.Version).Return(dogu, nil)

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		sut := NewDoguFetcher(client, nil, remoteDoguRegistry)

		// when
		fetchedDogu, cleanup, err := sut.FetchFromResource(context.Background(), doguCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, dogu, fetchedDogu)
		assert.Nil(t, cleanup)
		mock.AssertExpectationsForObjects(t, remoteDoguRegistry)
	})
}

func Test_doguFetcher_getDoguFromConfigMap(t *testing.T) {
	t.Run("fail as config map contains invalid json", func(t *testing.T) {
		// given
		sut := NewDoguFetcher(nil, nil, nil)
		redmineDoguDevelopmentMap := readDoguDescriptorConfigMap(t, redmineCrConfigMapBytes)
		redmineDoguDevelopmentMap.Data["dogu.json"] = "invalid dogu json"

		// when
		_, err := sut.getFromDevelopmentDoguMap(redmineDoguDevelopmentMap)

		// given
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal custom dogu descriptor")
	})
}

func Test_doguFetcher_getDoguFromRemoteRegistry(t *testing.T) {
	t.Run("fail when remote registry returns error", func(t *testing.T) {
		// given
		doguCr := readTestDataRedmineCr(t)

		remoteDoguRegistry := new(mocks3.Registry)
		remoteDoguRegistry.On("GetVersion", doguCr.Spec.Name, doguCr.Spec.Version).Return(nil, assert.AnError)

		sut := NewDoguFetcher(nil, nil, remoteDoguRegistry)

		// when
		_, err := sut.getDoguFromRemoteRegistry(doguCr)

		// then
		require.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to get dogu from remote dogu registry")
		mock.AssertExpectationsForObjects(t, remoteDoguRegistry)
	})
}

func Test_replaceK8sIncompatibleDoguDependencies(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given

		// when
		t.Fail()

		// then

	})
}
