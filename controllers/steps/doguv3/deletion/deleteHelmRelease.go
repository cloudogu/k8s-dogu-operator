package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	flux "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeleteHelmReleaseStep actively deletes the HelmRelease and waits (via requeue) until it is fully
// gone. The flux helm-controller runs the helm uninstall through its own finalizer on the
// HelmRelease, so the object only disappears once the uninstall completed.
type DeleteHelmReleaseStep struct {
	client        K8sClient
	eventRecorder EventRecorder
}

func NewDeleteHelmReleaseStep(client K8sClient, recorder EventRecorder) *DeleteHelmReleaseStep {
	return &DeleteHelmReleaseStep{client: client, eventRecorder: recorder}
}

func (s *DeleteHelmReleaseStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	release := &flux.HelmRelease{}
	err := s.client.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		// The HelmRelease is fully deleted - continue.
		return stepsv3.Continue()
	}
	if err != nil {
		return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to get HelmRelease %q: %w", doguResource.Spec.Name, err))
	}

	// Only issue the delete once; afterwards the object lingers with a deletion timestamp while flux
	// runs the uninstall, so we just keep waiting for it to vanish.
	if release.GetDeletionTimestamp().IsZero() {
		if dErr := s.client.Delete(ctx, release); dErr != nil {
			if apierrors.IsNotFound(dErr) {
				return stepsv3.Continue()
			}

			return failDeletion(s.eventRecorder, doguResource, fmt.Errorf("failed to delete HelmRelease %q: %w", release.Name, dErr))
		}

		log.FromContext(ctx).Info("issued deletion for HelmRelease", "name", release.Name)
	}

	// If flux reports the uninstall as stalled/failed, escalate (event + condition) but keep waiting:
	// the stall self-heals once the underlying issue clears, and giving up would leave the dogu stuck
	// in Terminating anyway.
	if message, stalled := helmReleaseUninstallStalled(release); stalled {
		return escalateDeletionStall(s.eventRecorder, doguResource, message)
	}

	return waitForDeletion("HelmRelease", release.Name)
}

// helmReleaseUninstallStalled reports whether flux signals that the HelmRelease's deletion cannot make
// progress without intervention: either the generic Stalled condition, or a Released condition whose
// reason is UninstallFailed.
func helmReleaseUninstallStalled(release *flux.HelmRelease) (string, bool) {
	if c := metautil.FindStatusCondition(release.Status.Conditions, fluxmeta.StalledCondition); c != nil && c.Status == metav1.ConditionTrue {
		return fmt.Sprintf("HelmRelease %q uninstall stalled: %s", release.Name, c.Message), true
	}

	if c := metautil.FindStatusCondition(release.Status.Conditions, flux.ReleasedCondition); c != nil && c.Reason == flux.UninstallFailedReason {
		return fmt.Sprintf("HelmRelease %q uninstall failed: %s", release.Name, c.Message), true
	}

	return "", false
}
