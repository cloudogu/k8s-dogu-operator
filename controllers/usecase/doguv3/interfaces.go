package doguv3

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationDetector interface {
	Detect(ctx context.Context, dogu *v3beta1.Dogu) (migration.State, error)
}

type Step interface {
	v3.Step
}

type K8sClient interface {
	client.Client
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}
