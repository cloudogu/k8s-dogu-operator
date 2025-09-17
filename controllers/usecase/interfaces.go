package usecase

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type Step interface {
	steps.Step
}
