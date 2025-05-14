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
		sut := newPodSpecBuilder(ldapDoguResource, ldapDogu).initContainers(&v1.Container{Image: testInitContainerImage})
		actual := sut.build()

		// then
		require.NotNil(t, actual)
		require.Len(t, actual.Spec.InitContainers, 1)
		assert.Equal(t, v1.Container{Image: testInitContainerImage}, actual.Spec.InitContainers[0])
	})
}

func Test_podSpecBuilder_sidecarContainers(t *testing.T) {
	ldapDoguResource := readLdapDoguResource(t)
	ldapDogu := readLdapDogu(t)

	t.Run("should not add sidecar container if list is empty", func(t *testing.T) {
		// when
		sut := newPodSpecBuilder(ldapDoguResource, ldapDogu).sidecarContainers(nil)
		actual := sut.build()

		// then
		require.NotNil(t, actual)
		assert.Len(t, actual.Spec.Containers, 1)
	})

	t.Run("should not add sidecar container if list is not empty", func(t *testing.T) {
		// when
		sut := newPodSpecBuilder(ldapDoguResource, ldapDogu).sidecarContainers(&v1.Container{Name: "exporter-sidecar", Image: "exporter:test"})
		actual := sut.build()

		// then
		require.NotNil(t, actual)
		assert.Len(t, actual.Spec.Containers, 2)
		assert.Equal(t, "exporter-sidecar", actual.Spec.Containers[1].Name)
		assert.Equal(t, "exporter:test", actual.Spec.Containers[1].Image)
	})
}
