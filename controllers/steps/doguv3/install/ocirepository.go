package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/dogu-lib/doguv3"
	doguv3reg "github.com/cloudogu/dogu-lib/doguv3/doguregistry"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	fluxShardingLabelKey   = "sharding.fluxcd.io/key"
	fluxShardingLabelValue = "ces"
)

const (
	containerRegistrySecret = "ces-container-registries"
)

// EnsureOCIRepositoryStep ensures the OCIRepository resource.
type EnsureOCIRepositoryStep struct {
	k8sClient     client.Client
	doguRegistry  DoguRegistryReader
	eventRecorder record.EventRecorder
}

func NewEnsureOCIRepositoryStep(k8sClient client.Client, doguRegistry DoguRegistryReader, recorder record.EventRecorder) *EnsureOCIRepositoryStep {
	return &EnsureOCIRepositoryStep{
		k8sClient:     k8sClient,
		doguRegistry:  doguRegistry,
		eventRecorder: recorder,
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
		err = fmt.Errorf("failed to get dogu descriptor for identifier %v: %w", identifier, err)
		if doguv3reg.IsGenericError(err) || doguv3reg.IsUnauthorizedError(err) || doguv3reg.IsForbiddenError(err) || doguv3reg.IsNotFoundError(err) {
			return stepsv3.Abort(v3beta1.ReasonDownloadFailed, err.Error())
		}
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	// Set OCIRepository metadata for lookup
	repository := &flux.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: doguResource.Namespace,
			Name:      doguResource.Spec.Name,
		},
	}

	// Use patch because the repository resource will be updated from the source-controller.
	// CreateOrUpdate would produce conflict errors and increase the number of reconciles.
	result, err := controllerutil.CreateOrPatch(ctx, eor.k8sClient, repository, func() error {
		if repository.Labels == nil {
			repository.Labels = make(map[string]string)
		}
		repository.Labels[fluxShardingLabelKey] = fluxShardingLabelValue
		repository.Labels[v3beta1.DoguLabelName] = doguResource.Spec.Name
		repository.Labels[v3beta1.DoguLabelVersion] = doguResource.Spec.Version
		repository.Spec = flux.OCIRepositorySpec{
			URL: dogu.Chart,
			Reference: &flux.OCIRepositoryRef{
				Tag: dogu.Version,
			},
			Interval:  metav1.Duration{Duration: time.Hour * 6},
			SecretRef: &meta.LocalObjectReference{Name: containerRegistrySecret},
		}

		return controllerutil.SetControllerReference(doguResource, repository, eor.k8sClient.Scheme())
	})

	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to createOrPatch OCIRepository %q: %w", doguResource.Spec.Name, err), v3beta1.ReasonInstalling)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		eor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository created")
		log.FromContext(ctx).Info("created OCIRepository", "name", repository.Name)
	case controllerutil.OperationResultUpdated:
		eor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository updated")
		log.FromContext(ctx).Info("updated OCIRepository", "name", repository.Name)

	}

	return stepsv3.Continue()
}
