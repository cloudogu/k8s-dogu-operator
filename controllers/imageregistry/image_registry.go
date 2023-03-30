package imageregistry

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/retry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const errMsgNoNetwork = "connect: network is unreachable"

var ImagePull = crane.Pull

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

// PullImageConfig pulls an image with the crane library. It uses basic auth for the registry authentication.
func (i *craneContainerImageRegistry) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	ctxOpt := crane.WithContext(ctx)
	authOpts := crane.WithAuth(&authn.Basic{
		Username: i.dockerUsername,
		Password: i.dockerPassword,
	})

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Try to pull image manifest from image: [%s]", image))

	var img imagev1.Image
	err := retry.OnError(15, retry.TestableRetryFunc, func() (err error) {
		img, err = ImagePull(image, authOpts, ctxOpt)
		if err != nil && strings.Contains(err.Error(), errMsgNoNetwork) {
			logger.Error(err, "Retry because the network is not reachable")
			return &retry.TestableRetrierError{}
		}

		return
	})

	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}

	return img.ConfigFile()
}
