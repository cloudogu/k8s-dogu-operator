package install

import (
	"context"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	doguv3steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DoguRegistryReader interface {
	Get(ctx context.Context, doguIdentifier doguv3.Identifier) (*doguv3.Dogu, error)
}

type ChartService interface {
	DoguMetaValues(ctx context.Context, doguResource *v3beta1.Dogu) ([]byte, bool, error)
	ValidateValues(ctx context.Context, doguResource *v3beta1.Dogu, values map[string]any) error
	Render(ctx context.Context, doguResource *v3beta1.Dogu, values map[string]any) ([]client.Object, error)
}

type ValueAssembler interface {
	Assemble(ctx context.Context, cr *v3beta1.Dogu, valuesMeta []byte) (values.Values, error)
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
