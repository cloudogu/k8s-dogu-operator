package migration

import (
	"context"

	v3beta1Dogu "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"k8s.io/apimachinery/pkg/runtime"
)

type MigrationDetector interface {
	Detect(ctx context.Context, dogu *v3beta1Dogu.Dogu) (migration.State, error)
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}
