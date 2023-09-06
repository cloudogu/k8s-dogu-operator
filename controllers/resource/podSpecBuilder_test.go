package resource

import (
	v1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_podSpecBuilder_initContainers(t *testing.T) {
	ldapDoguResource := readLdapDoguResource(t)
	ldapDogu := readLdapDogu(t)

	t.Run("should leave init container list empty", func(t *testing.T) {
		// when
		sut := newPodSpecBuilder(ldapDoguResource, ldapDogu).initContainers(nil)
		actual := sut.build()

		// then
		require.NotNil(t, actual)
		assert.Empty(t, actual.Spec.InitContainers)
	})
	t.Run("should set init container ", func(t *testing.T) {
		// when
		sut := newPodSpecBuilder(ldapDoguResource, ldapDogu).initContainers(&v1.Container{Image: testChownInitContainerImage})
		actual := sut.build()

		// then
		require.NotNil(t, actual)
		require.Len(t, actual.Spec.InitContainers, 1)
		assert.Equal(t, v1.Container{Image: testChownInitContainerImage}, actual.Spec.InitContainers[0])
	})
}
