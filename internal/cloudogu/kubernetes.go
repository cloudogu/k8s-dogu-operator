package cloudogu

import (
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/record"
)

type EcosystemInterface interface {
	ecoSystem.EcoSystemV1Alpha1Interface
}

type DoguInterface interface {
	ecoSystem.DoguInterface
}

type DeploymentInterface interface {
	appsv1client.DeploymentInterface
}

type EventRecorder interface {
	record.EventRecorder
}
