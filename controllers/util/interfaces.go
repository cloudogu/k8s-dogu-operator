package util

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
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

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	ecoSystem.EcoSystemV2Interface
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
