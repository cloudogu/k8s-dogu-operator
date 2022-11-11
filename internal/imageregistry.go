package internal

import (
	"context"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
)

type ImageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}
