package doguv2

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	doguv2.Step
}

type K8sClient interface {
	client.Client
}
