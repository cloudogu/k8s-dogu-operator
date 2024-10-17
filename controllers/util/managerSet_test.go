package util

import (
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/config"
)

func TestNewManagerSet(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		restConfig := &rest.Config{}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		clientSet := fake2.NewSimpleClientset()
		opConfig := &config.OperatorConfig{
			Namespace: "myNamespace",
		}
		ecosystemMock := newMockEcosystemInterface(t)
		applier := newMockApplier(t)
		var addImages map[string]string

		configRepos := ConfigRepositories{
			GlobalConfigRepository:  &repository.GlobalConfigRepository{},
			DoguConfigRepository:    &repository.DoguConfigRepository{},
			SensitiveDoguRepository: &repository.DoguConfigRepository{},
		}

		// when
		actual, err := NewManagerSet(restConfig, client, clientSet, ecosystemMock, opConfig, configRepos, applier, addImages)

		// then
		require.NoError(t, err)
		assert.NotNil(t, actual)

		assert.Equal(t, restConfig, actual.RestConfig)
		assert.NotNil(t, actual.CollectApplier)
		assert.NotNil(t, actual.FileExtractor)
		assert.NotNil(t, actual.CommandExecutor)
		assert.NotNil(t, actual.ServiceAccountCreator)
		assert.NotNil(t, actual.LocalDoguFetcher)
		assert.NotNil(t, actual.ResourceDoguFetcher)
		assert.NotNil(t, actual.ResourceUpserter)
		assert.NotNil(t, actual.DoguRegistrator)
		assert.NotNil(t, actual.ImageRegistry)
		assert.Equal(t, ecosystemMock, actual.EcosystemClient)
		assert.Equal(t, clientSet, actual.ClientSet)
		assert.NotNil(t, actual.DependencyValidator)
	})
}
