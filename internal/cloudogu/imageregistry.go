package cloudogu

import (
	"context"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
)

// ImageRegistry abstracts the use of a container registry and includes functionality to pull container images.
type ImageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}
