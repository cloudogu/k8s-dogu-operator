package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewShutdownHandler(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		doguInterfaceMock := newMockDoguInterface(t)

		// when
		handler := NewShutdownHandler(doguInterfaceMock)

		// then
		assert.Equal(t, doguInterfaceMock, handler.doguInterface)
	})

}
