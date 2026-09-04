package v2

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	v2.Step
}

type K8sClient interface {
	client.Client
}
