package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureOCIRepositoryStep ensures the OCIRepository resource.
type EnsureOCIRepositoryStep struct {
	k8sClient    client.Client
	doguRegistry DoguRegistryReader
}

func NewEnsureOCIRepositoryStep(k8sClient client.Client, doguRegistry DoguRegistryReader) *EnsureOCIRepositoryStep {
	return &EnsureOCIRepositoryStep{
		k8sClient:    k8sClient,
		doguRegistry: doguRegistry,
	}
}

func (eor *EnsureOCIRepositoryStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	// Query Dogu-Registry for the dogu descriptor
	identifier := doguv3.Identifier{
		DoguNamespace: doguResource.Spec.DoguNamespace,
		Name:          doguResource.Spec.Name,
		Version:       doguResource.Spec.Version,
	}
	dogu, err := eor.doguRegistry.Get(ctx, identifier)
	if err != nil {
		return stepsv3.StepResult{Err: fmt.Errorf("failed to get dogu descriptor for identifier %v: %w", identifier, err)}
	}

	// Set OCIRepository metadata for lookup
	repository := &flux.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: doguResource.Namespace,
			Name:      doguResource.Spec.Name,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, eor.k8sClient, repository, func() error {
		repository.Spec = flux.OCIRepositorySpec{
			URL: dogu.Chart,
			Reference: &flux.OCIRepositoryRef{
				Tag: dogu.Version,
			},
			SecretRef: &meta.LocalObjectReference{Name: "ces-container-registries"},
			// TODO: Interval and other options.
		}

		return controllerutil.SetControllerReference(doguResource, repository, eor.k8sClient.Scheme())
	})
	if err != nil {
		return stepsv3.StepResult{Err: fmt.Errorf("failed to createOrUpdate OCIRepository %q: %w", doguResource.Spec.Name, err)}
	}

	return stepsv3.Continue()
}
