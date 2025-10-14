package initfx

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDoguVersionRegistry(t *testing.T) {
	t.Run("should successfully create dogu version registry", func(t *testing.T) {
		// given
		cmInterface := newMockConfigMapInterface(t)

		// when
		registry := NewDoguVersionRegistry(cmInterface)

		// then
		assert.NotNil(t, registry)
	})
}

func TestNewLocalDoguDescriptorRepository(t *testing.T) {
	t.Run("should successfully create local dogu descriptor repository", func(t *testing.T) {
		// given
		cmInterface := newMockConfigMapInterface(t)

		// when
		repo := NewLocalDoguDescriptorRepository(cmInterface)

		// then
		assert.NotNil(t, repo)
	})
}

func TestNewLocalDoguFetcher(t *testing.T) {
	t.Run("should successfully create local dogu fetcher", func(t *testing.T) {
		// given
		cmInterface := newMockConfigMapInterface(t)
		registry := NewDoguVersionRegistry(cmInterface)
		repo := NewLocalDoguDescriptorRepository(cmInterface)

		// when
		fetcher := NewLocalDoguFetcher(registry, repo)

		// then
		assert.NotNil(t, fetcher)
	})
}

func Test_newRemoteDoguDescriptorRepository(t *testing.T) {
	t.Run("should successfully create remote dogu descriptor repository", func(t *testing.T) {
		// given
		operatorConfig := &config.OperatorConfig{}

		// when
		repo, err := newRemoteDoguDescriptorRepository(operatorConfig)

		// then
		assert.NotNil(t, repo)
		assert.NoError(t, err)
	})
}

func TestNewResourceDoguFetcher(t *testing.T) {
	t.Run("should successfully create remote dogu descriptor repository", func(t *testing.T) {
		// given
		operatorConfig := &config.OperatorConfig{}
		repo, err := newRemoteDoguDescriptorRepository(operatorConfig)
		require.NoError(t, err)
		clientMock := newMockK8sClient(t)

		// when
		remoteFetcher := NewResourceDoguFetcher(clientMock, repo)

		// then
		assert.NotNil(t, remoteFetcher)
	})
}
