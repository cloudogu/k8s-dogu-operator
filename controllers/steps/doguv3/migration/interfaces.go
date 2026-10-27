package migration

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MigrationDetector interface {
	Detect(ctx context.Context, dogu *v3beta1.Dogu) (migration.State, error)
}

type ResourceFinder interface {
	FindV2Resources(ctx context.Context, doguName string) ([]client.Object, error)
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}
