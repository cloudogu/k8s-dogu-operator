package imageregistry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
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
type craneContainerImageRegistry struct {
	// cache holds previously pulled image configs keyed by their image reference. It is nil when caching is disabled.
	cache *lruImageConfigCache
}

// NewCraneContainerImageRegistry creates a new instance of craneContainerImageRegistry. The cacheSize sizes the
// in-memory image config cache; a value <= 0 disables caching.
func NewCraneContainerImageRegistry(cacheSize int) ImageRegistry {
	return &craneContainerImageRegistry{cache: newImageConfigCache(cacheSize)}
}

// newImageConfigCache creates the image config cache. A non-positive size disables caching (returns nil).
func newImageConfigCache(size int) *lruImageConfigCache {
	if size <= 0 {
		return nil
	}

	return newLRUImageConfigCache(size)
}

// PullImageConfig pulls an image with the crane library. It uses basic auth for the registry authentication.
//
// If an image config cache is configured, the config for a given image reference is served from the cache instead of
// pulling it from the registry again. This avoids bursts of registry requests when the same image is inspected
// repeatedly (e.g. across reconcile retries or by several reconcile steps). The cached config must not be mutated by
// callers, as it is shared across all consumers of a cache hit.
func (i *craneContainerImageRegistry) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	logger := log.FromContext(ctx)

	if i.cache != nil {
		if cachedConfig, ok := i.cache.Get(image); ok {
			logger.Info(fmt.Sprintf("Using cached image config for image: [%s]", image))
			return cachedConfig, nil
		}
	}

	ctxOpt := crane.WithContext(ctx)

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

	stage, err := config.GetStage()
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get env var stage: %v", err))
	}

	var img imagev1.Image
	err = retry.OnErrorWithLimit(MaxWaitDuration, retry.AlwaysRetryFunc, func() (err error) {
		if stage == config.StageDevelopment {
			// The registry cannot be reached with the fqdn `k3ces.localdomain`. Therefore, the insecure flag is used.
			img, err = ImagePull(image, crane.WithAuthFromKeychain(authn.DefaultKeychain), crane.WithTransport(transport), ctxOpt, crane.Insecure)
		} else {
			img, err = ImagePull(image, crane.WithAuthFromKeychain(authn.DefaultKeychain), crane.WithTransport(transport), ctxOpt)
		}

		if err != nil {
			logger.Error(err, "error on image pull: retry")
			return err
		}

		return
	})

	if err != nil {
		return nil, fmt.Errorf("error pulling image: %w", err)
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}

	if i.cache != nil {
		i.cache.Add(image, configFile)
	}

	return configFile, nil
}
