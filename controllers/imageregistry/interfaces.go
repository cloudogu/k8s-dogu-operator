package imageregistry

import (
	"context"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
)

// ImageRegistry abstracts the use of a container registry and includes functionality to pull container images.
type ImageRegistry interface {
	// PullImageConfig is used to pull the config of the given container image.
	//
	// The returned config may be served from an in-memory cache and shared across callers, so it must be treated as
	// read-only; mutating it corrupts the cache and other callers' view.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}
