package controllers_test

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCraneContainerImageRegistry_PullImageConfig(t *testing.T) {
	imageRegistry := controllers.NewCraneContainerImageRegistry("user", "password")

	t.Run("successfully pulling image", func(t *testing.T) {
		server, src := setupCraneRegistry(t)
		defer server.Close()

		image, err := imageRegistry.PullImage(context.Background(), src)

		assert.NoError(t, err)
		assert.NotNil(t, image)
	})

	t.Run("error pulling image with wrong URL", func(t *testing.T) {
		_, err := imageRegistry.PullImage(context.Background(), "wrong url")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error pulling image")
	})
}

func setupCraneRegistry(t *testing.T) (*httptest.Server, string) {
	// Create local registry
	s := httptest.NewServer(registry.New())
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
