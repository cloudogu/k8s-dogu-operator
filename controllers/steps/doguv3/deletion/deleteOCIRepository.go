package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	flux "github.com/fluxcd/source-controller/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeleteOCIRepositoryStep actively deletes the OCIRepository and waits (via requeue) until it is fully gone.
type DeleteOCIRepositoryStep struct {
	client        K8sClient
	eventRecorder EventRecorder
}

func NewDeleteOCIRepositoryStep(client K8sClient, recorder EventRecorder) *DeleteOCIRepositoryStep {
	return &DeleteOCIRepositoryStep{client: client, eventRecorder: recorder}
}

func (s *DeleteOCIRepositoryStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	repository := &flux.OCIRepository{}
	err := s.client.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, repository)
	if apierrors.IsNotFound(err) {
		// The OCIRepository is fully deleted - continue.
		return stepsv3.Continue()
	}
	if err != nil {
		return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to get OCIRepository %q: %w", doguResource.Spec.Name, err))
	}

	if repository.GetDeletionTimestamp().IsZero() {
		if dErr := s.client.Delete(ctx, repository); dErr != nil {
			if apierrors.IsNotFound(dErr) {
				return stepsv3.Continue()
			}

			return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to delete OCIRepository %q: %w", repository.Name, dErr))
		}

		log.FromContext(ctx).Info("issued deletion for OCIRepository", "name", repository.Name)
	}

	return waitForDeletion("OCIRepository", repository.Name)
}
