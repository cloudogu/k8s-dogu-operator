package imageregistry_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	craneRegistry "github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
)

func TestCraneContainerImageRegistry_PullImageConfig(t *testing.T) {
	imageRegistry := imageregistry.NewCraneContainerImageRegistry("user", "password")

	t.Run("successfully pulling image", func(t *testing.T) {
		server, src := setupCraneRegistry(t)
		defer server.Close()

		image, err := imageRegistry.PullImageConfig(context.Background(), src)

		assert.NoError(t, err)
		assert.NotNil(t, image)
	})

	t.Run("error pulling image with wrong URL", func(t *testing.T) {
		_, err := imageRegistry.PullImageConfig(context.Background(), "wrong url")

		require.Error(t, err)
		assert.ErrorContains(t, err, "error pulling image")
	})
}

func setupCraneRegistry(t *testing.T) (*httptest.Server, string) {
	// Create local registry
	s := httptest.NewServer(craneRegistry.New())
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	src := fmt.Sprintf("%s/test/crane", u.Host)

	// Expected values.
	img, err := random.Image(1024, 5)
	if err != nil {
		t.Fatal(err)
	}

	// Load up the registry.
	if err := crane.Push(img, src); err != nil {
		t.Fatal(err)
	}

	return s, src
}
