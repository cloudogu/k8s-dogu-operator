package initfx

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
)

// NewImageRegistry creates the container image registry, sizing its image config cache from the operator config.
func NewImageRegistry(operatorConfig *config.OperatorConfig) imageregistry.ImageRegistry {
	return imageregistry.NewCraneContainerImageRegistry(operatorConfig.ImageConfigCacheSize)
}
