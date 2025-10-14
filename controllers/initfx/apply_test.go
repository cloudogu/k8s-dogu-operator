package initfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestNewCollectApplier(t *testing.T) {
	t.Run("should successfully create collect applier", func(t *testing.T) {
		// given
		restConfig := &rest.Config{}

		// when
		applier, err := NewCollectApplier(restConfig)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, applier)
	})
	t.Run("should fail to create collect applier", func(t *testing.T) {
		// given
		restConfig := &rest.Config{
			ExecProvider: &api.ExecConfig{},
			AuthProvider: &api.AuthProviderConfig{},
		}
		// when
		applier, err := NewCollectApplier(restConfig)

		// then
		assert.Error(t, err)
		assert.Empty(t, applier)
	})
}
