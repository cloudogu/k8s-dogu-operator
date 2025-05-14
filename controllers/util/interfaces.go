package util

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// dependencyValidator checks if all necessary dependencies for an upgrade are installed.
type dependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type securityValidator interface {
	// ValidateSecurity verifies the security fields of dogu descriptor and resource for correctness.
	ValidateSecurity(doguDescriptor *cesappcore.Dogu, doguResource *k8sv2.Dogu) error
}

type doguDataSeedValidator interface {
	ValidateDataSeeds(ctx context.Context, doguDescriptor *cesappcore.Dogu, doguResource *k8sv2.Dogu) error
}

type dataSeederInitContainerGenerator interface {
	BuildDataSeederContainer(dogu *cesappcore.Dogu, doguResource *k8sv2.Dogu, image string, requirements coreV1.ResourceRequirements) (*coreV1.Container, error)
}

// requirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type requirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (coreV1.ResourceRequirements, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
}

// applier provides ways to apply unstructured Kubernetes resources against the API.
//
//nolint:unused
//goland:noinspection GoUnusedType
type applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type clientSet interface {
	kubernetes.Interface
}
