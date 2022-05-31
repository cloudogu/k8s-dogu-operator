package controllers

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_templateNamespaces(t *testing.T) {
	t.Run("should template namespace", func(t *testing.T) {
		tempDoc := []byte(`hello {{ .Namespace }}`)

		actual, err := templateNamespaces(tempDoc, "world")

		require.NoError(t, err)
		expected := []byte(`hello world`)
		assert.Equal(t, expected, actual)
	})

	t.Run("should return error", func(t *testing.T) {
		tempDoc := []byte(`hello {{ .Namespace `)

		_, err := templateNamespaces(tempDoc, "world")

		require.Error(t, err)
	})
}
