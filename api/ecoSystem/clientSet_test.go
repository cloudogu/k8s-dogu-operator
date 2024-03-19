package ecoSystem

import (
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
	"testing"
)

func TestNewForConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		config := &rest.Config{}

		// when
		clientSet, err := NewForConfig(config)

		// then
		require.NoError(t, err)
		require.NotNil(t, clientSet)
	})

}

func TestEcoSystemV1Alpha1Client_Dogus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		config := &rest.Config{}
		clientSet, err := NewForConfig(config)
		require.NoError(t, err)
		require.NotNil(t, clientSet)

		// when
		client := clientSet.Dogus("ecosystem")

		// then
		require.NotNil(t, client)
	})
}
