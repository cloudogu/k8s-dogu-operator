package install

import (
	"context"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	doguv3steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DoguRegistryReader interface {
	Get(ctx context.Context, doguIdentifier doguv3.Identifier) (*doguv3.Dogu, error)
}

type K8sClient interface {
	client.Client
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

type Step interface {
	doguv3steps.Step
}

type valueAssembler interface {
	Assemble(ctx context.Context, cr *v3beta1.Dogu, patchTpl []byte) (values3.Values, error)
}
