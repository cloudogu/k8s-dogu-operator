package controllers

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCraneContainerImageRegistry_PullImageConfig(t *testing.T) {

	t.Run("successful pulling image", func(t *testing.T) {
		server, src := setupCraneRegistry(t)
		defer server.Close()

		imageRegistry := NewCraneContainerImageRegistry("user", "password")

		imageConfig, err := imageRegistry.PullImageConfig(context.Background(), src)

		assert.NoError(t, err)
		assert.NotNil(t, imageConfig)
	})

	t.Run("error pulling image", func(t *testing.T) {
		server, src := setupCraneRegistry(t)
		server.Close()

		imageRegistry := NewCraneContainerImageRegistry("user", "password")

		imageConfig, err := imageRegistry.PullImageConfig(context.Background(), src)

		assert.Error(t, err)
		assert.Nil(t, imageConfig)
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
