package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	flux "github.com/fluxcd/helm-controller/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// UnsuspendHelmReleaseStep clears a HelmRelease's suspend flag before deletion. A suspended
// HelmRelease is ignored by the flux helm-controller, so its uninstall (and thus garbage collection
// of the helm-managed resources) would never run. Setting suspend=false lets the helm-controller
// process the subsequent deletion.
type UnsuspendHelmReleaseStep struct {
	client        K8sClient
	eventRecorder EventRecorder
}

func NewUnsuspendHelmReleaseStep(client K8sClient, recorder EventRecorder) *UnsuspendHelmReleaseStep {
	return &UnsuspendHelmReleaseStep{client: client, eventRecorder: recorder}
}

func (s *UnsuspendHelmReleaseStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	release := &flux.HelmRelease{}
	err := s.client.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		// Nothing to unsuspend - the HelmRelease is already gone.
		return stepsv3.Continue()
	}
	if err != nil {
		return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to get HelmRelease %q: %w", doguResource.Spec.Name, err))
	}

	if !release.Spec.Suspend {
		return stepsv3.Continue()
	}

	release.Spec.Suspend = false
	if uErr := s.client.Update(ctx, release); uErr != nil {
		return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to unsuspend HelmRelease %q: %w", release.Name, uErr))
	}

	log.FromContext(ctx).Info("unsuspended HelmRelease for deletion", "name", release.Name)
	return stepsv3.Continue()
}
