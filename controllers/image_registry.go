package controllers

import (
	"context"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// CraneContainerImageRegistry is a component to interact with a container registry.
// It is able to pull the config of an image and uses the crane library
type CraneContainerImageRegistry struct {
	dockerUsername string
	dockerPassword string
}

// NewCraneContainerImageRegistry creates a new instance of CraneContainerImageRegistry
func NewCraneContainerImageRegistry(dockerUsername string, dockerPassword string) *CraneContainerImageRegistry {
	return &CraneContainerImageRegistry{
		dockerUsername: dockerUsername,
		dockerPassword: dockerPassword,
	}
}

// PullImage pulls an image with the crane library. It uses basic auth for the registry authentication
func (i *CraneContainerImageRegistry) PullImage(ctx context.Context, image string) (v1.Image, error) {
	ctxOpt := crane.WithContext(ctx)
	authOpts := crane.WithAuth(&authn.Basic{
		Username: i.dockerUsername,
		Password: i.dockerPassword,
	})
	img, err := crane.Pull(image, authOpts, ctxOpt)
	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}
	return img, nil
}
