package upgrade

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	cesregistry.LocalDoguFetcher
}

// resourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type resourceDoguFetcher interface {
	cesregistry.ResourceDoguFetcher
}

// DependencyValidator checks if all necessary dependencies for an upgrade are installed.
type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

// doguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type doguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

type k8sClient interface {
	client.Client
}

type execPodFactory interface {
	exec.ExecPodFactory
}

type resourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except deploymentPatch which may be nil.
	// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
	UpsertDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu, deploymentPatch func(*apps.Deployment)) (*apps.Deployment, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type securityContextGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu, doguResource *v2.Dogu) (*corev1.PodSecurityContext, *corev1.SecurityContext)
}

type commandExecutor interface {
	exec.CommandExecutor
}

//nolint:unused
//goland:noinspection GoUnusedType
type resourceGenerator interface {
	resource.DoguResourceGenerator
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
}
