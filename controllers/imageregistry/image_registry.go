package imageregistry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cloudogu/retry-lib/retry"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ImagePull       = crane.Pull
	MaxWaitDuration = time.Minute * 1
)

// craneContainerImageRegistry is a component to interact with a container registry.
// It is able to pull the config of an image and uses the crane library
type craneContainerImageRegistry struct{}

// NewCraneContainerImageRegistry creates a new instance of craneContainerImageRegistry
func NewCraneContainerImageRegistry() ImageRegistry {
	return &craneContainerImageRegistry{}
}

// PullImageConfig pulls an image with the crane library. It uses basic auth for the registry authentication.
func (i *craneContainerImageRegistry) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	ctxOpt := crane.WithContext(ctx)

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Try to pull image manifest from image: [%s]", image))

	transport := remote.DefaultTransport
	proxyURL, found := os.LookupEnv("PROXY_URL")
	if found && len(proxyURL) > 0 {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}

		t, ok := transport.(*http.Transport)
		if !ok {
			return nil, errors.New("type assertion error: no transport")
		}
		t.Proxy = http.ProxyURL(parsedURL)
	}

	var img imagev1.Image
	err := retry.OnErrorWithLimit(MaxWaitDuration, retry.AlwaysRetryFunc, func() (err error) {
		img, err = ImagePull(image, crane.WithAuthFromKeychain(authn.DefaultKeychain), crane.WithTransport(transport), ctxOpt)
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
