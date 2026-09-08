package doguv3

import (
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
)

type Step interface {
	v3.Step
}
