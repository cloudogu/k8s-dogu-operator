package imageregistry_test

import (
	"context"
	"fmt"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	craneRegistry "github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
)

func TestCraneContainerImageRegistry_PullImageConfig(t *testing.T) {
	imageRegistry := imageregistry.NewCraneContainerImageRegistry()

	t.Run("successfully pulling image", func(t *testing.T) {
		server, src := setupCraneRegistry(t)
		defer server.Close()

		image, err := imageRegistry.PullImageConfig(context.Background(), src)

		assert.NoError(t, err)
		assert.NotNil(t, image)
	})

	t.Run("error pulling image with wrong URL", func(t *testing.T) {
		oldMaxWaitDuration := imageregistry.MaxWaitDuration
		imageregistry.MaxWaitDuration = time.Second * 3
		defer func() {
			imageregistry.MaxWaitDuration = oldMaxWaitDuration
		}()
		_, err := imageRegistry.PullImageConfig(context.Background(), "wrong url")

		require.Error(t, err)
		assert.ErrorContains(t, err, "error pulling image")
	})

	t.Run("should retry when an error occurs", func(t *testing.T) {
		// given
		oldMaxWaitDuration := imageregistry.MaxWaitDuration
		imageregistry.MaxWaitDuration = time.Second * 3
		defer func() {
			imageregistry.MaxWaitDuration = oldMaxWaitDuration
		}()

		oldImagePull := imageregistry.ImagePull
		i := 0
		imageregistry.ImagePull = func(src string, opt ...crane.Option) (imagev1.Image, error) {
			if i < 1 {
				i++
				return nil, assert.AnError
			}
			return mockImage{}, nil
		}

		defer func() {
			imageregistry.ImagePull = oldImagePull
		}()

		// when
		_, err := imageRegistry.PullImageConfig(context.Background(), "dummyImage")

		// then
		assert.NoError(t, err)
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

type mockImage struct {
	imagev1.Image
}

func (mi mockImage) ConfigFile() (*imagev1.ConfigFile, error) {
	return nil, nil
}
