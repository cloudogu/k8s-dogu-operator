package initfx

import (
	"testing"

	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewHostAliasGenerator(t *testing.T) {
	t.Run("should successfully create host alias generator", func(t *testing.T) {
		// given
		repo := repository.GlobalConfigRepository{}

		// when
		generator := NewHostAliasGenerator(repo)

		// then
		assert.NotNil(t, generator)
	})
}
