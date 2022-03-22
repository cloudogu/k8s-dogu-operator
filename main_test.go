package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_getWatchNamespace(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expectedNamespace := "default"
		t.Setenv("WATCH_NAMESPACE", expectedNamespace)
		namespace, err := getWatchNamespace()
		require.NoError(t, err)

		assert.Equal(t, expectedNamespace, namespace)
	})
	t.Run("fail", func(t *testing.T) {
		_, err := getWatchNamespace()
		require.Error(t, err)
	})

}
