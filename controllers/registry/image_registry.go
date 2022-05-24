package registry

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
)

// craneContainerImageRegistry is a component to interact with a container registry.
// It is able to pull the config of an image and uses the crane library
type craneContainerImageRegistry struct {
	dockerUsername string
	dockerPassword string
}

// NewCraneContainerImageRegistry creates a new instance of craneContainerImageRegistry
func NewCraneContainerImageRegistry(dockerUsername string, dockerPassword string) *craneContainerImageRegistry {
	return &craneContainerImageRegistry{
		dockerUsername: dockerUsername,
		dockerPassword: dockerPassword,
	}
}

// PullImageConfig pulls a image with the crane library. It uses basic auth for the registry authentication
func (i *craneContainerImageRegistry) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	ctxOpt := crane.WithContext(ctx)
	authOpts := crane.WithAuth(&authn.Basic{
		Username: i.dockerUsername,
		Password: i.dockerPassword,
	})

	img, err := crane.Pull(image, authOpts, ctxOpt)
	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}

	return img.ConfigFile()
}
