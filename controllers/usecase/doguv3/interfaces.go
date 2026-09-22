package doguv3

import (
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Step interface {
	v3.Step
}

type K8sClient interface {
	client.Client
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}
