package postinstall

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
)

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

// exportManager includes functionality to handle the export flag for dogus in the cluster.
type exportManager interface {
	// UpdateExportMode activates/deactivates the export mode for the dogu
	UpdateExportMode(ctx context.Context, doguResource *v2.Dogu) error
}

// supportManager includes functionality to handle the support flag for dogus in the cluster.
type supportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error)
}

type doguConfigRepository interface {
	Get(ctx context.Context, name cescommons.SimpleName) (config.DoguConfig, error)
	Create(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Delete(ctx context.Context, name cescommons.SimpleName) error
	Watch(ctx context.Context, dName cescommons.SimpleName, filters ...config.WatchFilter) (<-chan repository.DoguConfigWatchResult, error)
	SetOwnerReference(ctx context.Context, dName cescommons.SimpleName, owners []metav1.OwnerReference) error
}

type additionalMountManager interface {
	AdditionalMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error)
	UpdateAdditionalMounts(ctx context.Context, doguResource *v2.Dogu) error
}

type doguRestartManager interface {
	RestartAllDogus(ctx context.Context) error
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}
