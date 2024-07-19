package util

import (
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

func TestNewManagerSet(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		restConfig := &rest.Config{}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		clientSet := fake2.NewSimpleClientset()
		opConfig := &config.OperatorConfig{
			Namespace: "myNamespace",
		}
		ecosystemMock := mocks.NewEcosystemInterface(t)
		applier := mocks.NewApplier(t)
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
	})
}
