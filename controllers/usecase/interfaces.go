package usecase

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	steps.Step
}

type K8sClient interface {
	client.Client
}
