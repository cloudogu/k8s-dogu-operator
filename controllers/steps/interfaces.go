package steps

import appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}
