package initfx

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
)

// NewImageRegistry creates the container image registry, sizing its image config cache from the operator config.
// It is a var so integration tests can override it with a mock.
var NewImageRegistry = func(operatorConfig *config.OperatorConfig) imageregistry.ImageRegistry {
	return imageregistry.NewCraneContainerImageRegistry(operatorConfig.ImageConfigCacheSize)
}
