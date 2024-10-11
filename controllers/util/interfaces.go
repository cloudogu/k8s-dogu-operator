package util

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DependencyValidator checks if all necessary dependencies for an upgrade are installed.
type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type EcosystemInterface interface {
	ecoSystem.EcoSystemV1Alpha1Interface
}

// Applier provides ways to apply unstructured Kubernetes resources against the API.
type Applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type ClientSet interface {
	kubernetes.Interface
}
