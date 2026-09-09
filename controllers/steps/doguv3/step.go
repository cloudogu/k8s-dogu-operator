package doguv3

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

// Step defines a single aspect in a dogu lifecycle phase which usually ends up in changed side-effects.
//
// The sequence of which step is ran by the reconciler is fixed. Anyhow, they are supposed to work in an idempotent fashion,
// so that the step exits early without error if the desired side effect was already accomplished or is going to re-run upon an early error.
//
// Any means for cluster or any other service access must be provided during the step creation.
type Step interface {
	// Run executes the step and returns a StepResult indicating any errors and whether a follow-up requeue is in order to continue the reconcilication.
	Run(ctx context.Context, resource *v3beta1.Dogu) StepResult
}

// StepResult indicates if and when a follow-up requeue is in order to continue the reconcilication.
type StepResult struct {
	// RequeueAfter contains the earliest duration when the Kubernetes API restarts a new reconciliation loop for the current dogu.
	RequeueAfter time.Duration
	// Erro contains an error if one occurred.
	Err error
	// Continue with a value of true flags if another requeue should be conducted. Otherwise, the dogu wil no longer be reconciled. This field will be evaluated after checks for RequeueAfter or Err.
	Continue bool
	// ReadyReason contains the reason for the ready condition.
	// We add the information to the StepResult in v3 because the reonciler should set the ready condition in a central place.
	// The steps should set other specific conditions.
	ReadyReason string
	// ReadyMessage contains the message for the ready condition.
	ReadyMessage string
}

// RequeueAfter is a convenience function returning a StepResult with the given duration, indicating a requeue without further error.
func RequeueAfter(requeueAfter time.Duration, reason, msg string) StepResult {
	return StepResult{
		RequeueAfter: requeueAfter,
		ReadyReason:  reason,
		ReadyMessage: msg,
	}
}

// Continue is a convenience function returning a StepResult that indicates to positively continue with the next step of the dogu phase.
//
// Its counterpart is Abort() for stopping the dogu's reconciliation.
func Continue() StepResult {
	return StepResult{
		Continue: true,
	}
}

// Abort is a convenience function returning a StepResult that indicates to end this step and the reconciliation in the dogu's lifecycle in general.
//
// Its counterpart is Continue() for continuing the dogu's reconciliation.
func Abort(reason, msg string) StepResult {
	return StepResult{
		Continue:     false,
		ReadyReason:  reason,
		ReadyMessage: msg,
	}
}

// RequeueWithError is a convenience function returning a StepResult that indicates to end this step.
//
// In contrast to Abort(), this function keeps up the dogu's reconciliation.
func RequeueWithError(err error, reason string) StepResult {
	return StepResult{
		Err:          err,
		ReadyMessage: reason,
		// Message will be derived from the error
	}
}
