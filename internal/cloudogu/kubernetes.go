package cloudogu

import (
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
)

type EcosystemInterface interface {
	ecoSystem.EcoSystemV1Alpha1Interface
}

type DoguInterface interface {
	ecoSystem.DoguInterface
}

type DoguRestartInterface interface {
	ecoSystem.DoguRestartInterface
}
