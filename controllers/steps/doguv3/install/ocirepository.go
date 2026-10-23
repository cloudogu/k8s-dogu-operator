package install

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudogu/dogu-lib/doguv3"
	doguv3reg "github.com/cloudogu/dogu-lib/doguv3/doguregistry"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pending"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	labelKeyFluxSharding = "sharding.fluxcd.io/key"
)

const (
	containerRegistrySecret = "ces-container-registries"
)

// EnsureOCIRepositoryStep ensures the OCIRepository resource.
type EnsureOCIRepositoryStep struct {
	k8sClient     K8sClient
	doguRegistry  DoguRegistryReader
	eventRecorder EventRecorder
}

func NewEnsureOCIRepositoryStep(k8sClient K8sClient, doguRegistry DoguRegistryReader, recorder EventRecorder) *EnsureOCIRepositoryStep {
	return &EnsureOCIRepositoryStep{
		k8sClient:     k8sClient,
		doguRegistry:  doguRegistry,
		eventRecorder: recorder,
	}
}

// PendingChange is a placeholder until the OCI comparison is implemented in the pause/resume story.
func (eor *EnsureOCIRepositoryStep) PendingChange(ctx context.Context, dogu *v3beta1.Dogu) (pending.PendingChange, error) {
	return pending.PendingChange{Status: metav1.ConditionUnknown, Message: "The chart source comparison is not implemented yet"}, nil
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
		err = fmt.Errorf("failed to get dogu descriptor for identifier %s: %w", identifier, err)
		if doguv3reg.IsGenericError(err) || doguv3reg.IsUnauthorizedError(err) || doguv3reg.IsForbiddenError(err) || doguv3reg.IsNotFoundError(err) {
			return stepsv3.Abort(v3beta1.ReasonDownloadFailed, err.Error())
		}
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	// Set OCIRepository metadata for lookup
	repository := &flux.OCIRepository{
		Namespace: doguResource.Namespace,
		Name:      doguResource.Spec.Name,
	}

	if doguResource.Spec.PauseReconciliation {
		// if PauseReconciliation is set, we don't want to create/update the repository resource, but exit here
		return eor.checkForPendingChanges(ctx, dogu, repository, doguResource)
	}

	// Use patch because the repository resource will be updated from the source-controller.
	// CreateOrUpdate would produce conflict errors and increase the number of reconciles.
	result, err := controllerutil.CreateOrPatch(ctx, eor.k8sClient, repository, func() error {
		configureRepository(repository, doguResource, dogu)
		return controllerutil.SetControllerReference(doguResource, repository, eor.k8sClient.Scheme())
	})

	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to createOrPatch OCIRepository %q: %w", doguResource.Spec.Name, err), v3beta1.ReasonInstalling)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		eor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository created")
		logging.Info(ctx, "created OCIRepository", "name", repository.Name)
	case controllerutil.OperationResultUpdated:
		eor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, "OCIRepository updated")
		logging.Info(ctx, "updated OCIRepository", "name", repository.Name)

	}

	return stepsv3.Continue()
}

func configureRepository(repository *flux.OCIRepository, doguResource *v3beta1.Dogu, dogu *doguv3.Dogu) {
	if repository.Labels == nil {
		repository.Labels = make(map[string]string)
	}
	repository.Labels[labelKeyFluxSharding] = resource.LabelValueCes
	repository.Labels[v3beta1.DoguLabelName] = doguResource.Spec.Name
	repository.Labels[v3beta1.DoguLabelVersion] = doguResource.Spec.Version
	repository.Spec = flux.OCIRepositorySpec{
		URL: dogu.Chart,
		Reference: &flux.OCIRepositoryRef{
			Tag: dogu.Version,
		},
		Interval:  metav1.Duration{Duration: time.Hour * 6},
		SecretRef: &meta.LocalObjectReference{Name: containerRegistrySecret},
		Provider:  flux.GenericOCIProvider,
		Timeout:   &metav1.Duration{Duration: 60 * time.Second},
	}
}

func (eor *EnsureOCIRepositoryStep) checkForPendingChanges(ctx context.Context, dogu *doguv3.Dogu, repository *flux.OCIRepository, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	err := eor.k8sClient.Get(ctx, client.ObjectKey{Namespace: repository.Namespace, Name: repository.Name}, repository)
	if err != nil {
		if !errors.IsNotFound(err) {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonReconciliationPaused)
		}
	}

	desiredRepo := repository.DeepCopy()
	configureRepository(desiredRepo, doguResource, dogu)
	if !checkForChanges(repository, desiredRepo) {
		return stepsv3.Continue()
	}
	changeCondition := metav1.Condition{
		Type:               v3beta1.ConditionUpdatePending,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: doguResource.Generation,
		Reason:             v3beta1.ReasonReconciliationPaused,
		Message:            "Changes not propagated to OCIRepository, because dogu reconciliation is paused",
	}
	if metautil.SetStatusCondition(&doguResource.Status.Conditions, changeCondition) {
		err := eor.k8sClient.Status().Update(ctx, doguResource)
		if err != nil {
			if errors.IsConflict(err) {
				return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, err.Error())
			}
			return stepsv3.RequeueWithError(err, v3beta1.ReasonReconciliationPaused)
		}
		eor.eventRecorder.Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionUpdatePending, "Changes not propagated to OCIRepository, because dogu reconciliation is paused")
		logging.Info(ctx, "Changes not propagated to OCIRepository, because dogu reconciliation is paused", "dogu", doguResource.Name)
		return stepsv3.Continue()
	}
	return stepsv3.Continue()
}

func checkForChanges(actual *flux.OCIRepository, desired *flux.OCIRepository) bool {
	if !reflect.DeepEqual(actual.Spec, desired.Spec) {
		return true
	}
	for key := range desired.Labels {
		if actual.Labels[key] != desired.Labels[key] {
			return true
		}
	}
	return false
}
