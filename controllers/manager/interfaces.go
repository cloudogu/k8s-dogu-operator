package manager

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	coreV1 "k8s.io/api/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
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

type podInterface interface {
	v1.PodInterface
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}
type additionalMountsInitContainerGenerator interface {
	BuildAdditionalMountInitContainer(ctx context.Context, dogu *cesappcore.Dogu, doguResource *v2.Dogu, image string, requirements coreV1.ResourceRequirements) (*coreV1.Container, error)
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
