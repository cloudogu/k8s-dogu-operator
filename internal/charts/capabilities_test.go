package charts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakediscovery "k8s.io/client-go/discovery/fake"
	coretesting "k8s.io/client-go/testing"
)

func Test_discoverVersionSet(t *testing.T) {
	t.Run("derives versions from discovered groups and resources", func(t *testing.T) {
		fakeDiscovery := &fakediscovery.FakeDiscovery{
			Fake: &coretesting.Fake{
				Resources: []*metav1.APIResourceList{
					{
						GroupVersion: "apps/v1",
						APIResources: []metav1.APIResource{{Kind: "Deployment"}},
					},
					{
						GroupVersion: "batch/v1",
						APIResources: []metav1.APIResource{{Kind: "Job"}},
					},
				},
			},
		}

		versions, err := discoverVersionSet(fakeDiscovery)

		require.NoError(t, err)
		// Both the plain GroupVersion and the GroupVersion/Kind are registered, mirroring Helm.
		assert.True(t, versions.Has("apps/v1"))
		assert.True(t, versions.Has("apps/v1/Deployment"))
		assert.True(t, versions.Has("batch/v1"))
		assert.True(t, versions.Has("batch/v1/Job"))
	})

	t.Run("falls back to the default version set when nothing is discovered", func(t *testing.T) {
		fakeDiscovery := &fakediscovery.FakeDiscovery{Fake: &coretesting.Fake{}}

		versions, err := discoverVersionSet(fakeDiscovery)

		require.NoError(t, err)
		assert.Equal(t, chartutil.DefaultVersionSet, versions)
	})
}
