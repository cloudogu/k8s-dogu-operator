package manager

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	coreV1 "k8s.io/api/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
}

type DoguRestartManager interface {
	RestartDogu(ctx context.Context, dogu *v2.Dogu) error
}

type DoguExportManager interface {
	UpdateExportMode(ctx context.Context, doguResource *v2.Dogu) error
}

// SupportManager includes functionality to handle the support flag for dogus in the cluster.
type SupportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error)
}

type podInterface interface {
	v1.PodInterface
}

type doguInterface interface {
	doguClient.DoguInterface
}

type resourceUpserter interface {
	resource.ResourceUpserter
}

type eventRecorder interface {
	record.EventRecorder
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

type AdditionalMountManager interface {
	AdditionalMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error)
	UpdateAdditionalMounts(ctx context.Context, doguResource *v2.Dogu) error
}

type resourceGenerator interface {
	resource.DoguResourceGenerator
}

// requirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
//
//nolint:unused
//goland:noinspection GoUnusedType
type requirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (coreV1.ResourceRequirements, error)
}

type doguAdditionalMountsValidator interface {
	ValidateAdditionalMounts(ctx context.Context, doguDescriptor *cesappcore.Dogu, doguResource *v2.Dogu) error
}

type k8sClient interface {
	client.Client
}
