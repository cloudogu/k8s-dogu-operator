package deletion

import (
	doguv3steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type K8sClient interface {
	client.Client
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

type Step interface {
	doguv3steps.Step
}
