package imageregistry

import (
	"context"
	"fmt"
	"github.com/cloudogu/retry-lib/retry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var (
	ImagePull       = crane.Pull
	MaxWaitDuration = time.Minute * 1
)

// craneContainerImageRegistry is a component to interact with a container registry.
// It is able to pull the config of an image and uses the crane library
type craneContainerImageRegistry struct{}

// NewCraneContainerImageRegistry creates a new instance of craneContainerImageRegistry
func NewCraneContainerImageRegistry() *craneContainerImageRegistry {
	return &craneContainerImageRegistry{}
}

// PullImageConfig pulls an image with the crane library. It uses basic auth for the registry authentication.
func (i *craneContainerImageRegistry) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	ctxOpt := crane.WithContext(ctx)

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Try to pull image manifest from image: [%s]", image))

	var img imagev1.Image
	err := retry.OnErrorWithLimit(MaxWaitDuration, retry.AlwaysRetryFunc, func() (err error) {
		img, err = ImagePull(image, crane.WithAuthFromKeychain(authn.DefaultKeychain), ctxOpt)
		if err != nil {
			logger.Error(err, "error on image pull: retry")
			return err
		}

		return
	})

	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}

	return img.ConfigFile()
}
