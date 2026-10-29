package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReasonValid is the reason for ConditionValid when the dogu's values satisfy the chart schema
	// (if checked) and the chart renders successfully.
	ReasonValid = "Valid"
	// ReasonRenderFailed is the reason for ConditionValid when the chart cannot be rendered with the assembled values.
	ReasonRenderFailed = "RenderFailed"
)

// ValidateChartStep validates a dogu's chart before any HelmRelease is written. It assembles the
// merged values, optionally validates them against the chart's values.schema.json (honoring
// Spec.SkipSchemaValidation) and renders the chart server-side without creating or modifying any
// cluster resource. A chart whose values violate the schema, that fails to render or that would
// make a forbidden pvc change (shrink or storage class change) aborts the reconciliation via
// ConditionValid so no HelmRelease is produced.
type ValidateChartStep struct {
	chartService  chartService
	assembler     valueAssembler
	resizeChecker ResizeChecker
	k8sClient     K8sClient
}

func NewValidateChartStep(chartService chartService, assembler valueAssembler, resizeChecker ResizeChecker, k8sClient K8sClient) *ValidateChartStep {
	return &ValidateChartStep{
		chartService:  chartService,
		assembler:     assembler,
		resizeChecker: resizeChecker,
		k8sClient:     k8sClient,
	}
}

func (vcs *ValidateChartStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		logging.Debug(ctx, "Skipping ValidateChartStep because pausedReconciliation is set for dogu", "dogu", doguResource.Name)
		return stepsv3.Continue()
	}

	valuesMeta, _, err := vcs.chartService.DoguMetaValues(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get dogu meta values for dogu %s/%s: %w", doguResource.Namespace, doguResource.Name, err), v3beta1.ReasonInstalling)
	}

	runtimePatchTemplate, found, err := vcs.chartService.RuntimePatchTemplate(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to retrieve runtime patch template for dogu %s/%s: %w", doguResource.Namespace, doguResource.Name, err), v3beta1.ReasonInstalling)
	}

	isEmpty := len(bytes.TrimSpace(runtimePatchTemplate)) == 0
	emptyPatchFileExists := found && isEmpty
	if emptyPatchFileExists {
		err := fmt.Errorf("%s must not be empty for dogu %s/%s", values.RuntimePatchTemplateFileName, doguResource.Namespace, doguResource.Name)
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	assembledValues, err := vcs.assembler.Assemble(ctx, doguResource, valuesMeta, runtimePatchTemplate)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to assemble values for dogu %s/%s: %w", doguResource.Namespace, doguResource.Name, err), v3beta1.ReasonInstalling)
	}

	// Record whether schema validation is skipped and, unless skipped, validate the values.
	schemaSkippedChanged := vcs.setSchemaValidationSkippedCondition(doguResource)
	if !doguResource.Spec.SkipSchemaValidation {
		if vErr := vcs.chartService.ValidateValues(ctx, doguResource, assembledValues); vErr != nil {
			msg := fmt.Sprintf("values do not satisfy the chart schema: %v", vErr)

			return vcs.abort(ctx, doguResource, schemaSkippedChanged, v3beta1.ReasonSchemaInvalid, msg)
		}

		logging.Debug(ctx, "Successfully validated values against schema for dogu", "dogu", doguResource.Spec.Name)
	}

	rendered, rErr := vcs.chartService.Render(ctx, doguResource, assembledValues)
	if rErr != nil {
		msg := fmt.Sprintf("chart could not be rendered: %v", rErr)
		return vcs.abort(ctx, doguResource, schemaSkippedChanged, ReasonRenderFailed, msg)
	}

	logging.Debug(ctx, "Successfully rendered dogu helm chart", "dogu", doguResource.Spec.Name)

	if _, cErr := vcs.resizeChecker.Check(ctx, doguResource.Namespace, rendered); cErr != nil {
		if reason, rejected := volumeRejectionReason(cErr); rejected {
			return vcs.abort(ctx, doguResource, schemaSkippedChanged, reason, fmt.Sprintf("volumes cannot be changed as requested: %v", cErr))
		}

		return stepsv3.RequeueWithError(fmt.Errorf("failed to check volumes: %w", cErr), v3beta1.ReasonInstalling)
	}

	return vcs.succeed(ctx, doguResource, schemaSkippedChanged)
}

func volumeRejectionReason(err error) (string, bool) {
	var shrinkErr *pvc.VolumeShrinkError
	if errors.As(err, &shrinkErr) {
		return v3beta1.ReasonVolumeShrink, true
	}

	var storageClassErr *pvc.StorageClassImmutableError
	if errors.As(err, &storageClassErr) {
		return v3beta1.ReasonStorageClassImmutable, true
	}

	return "", false
}

// setSchemaValidationSkippedCondition mirrors the Spec.SkipSchemaValidation flag onto the
// ConditionSchemaValidationSkipped condition. It only mutates the in-memory resource; the change is
// persisted together with ConditionValid when the step reaches an exit point. It reports whether the condition changed.
func (vcs *ValidateChartStep) setSchemaValidationSkippedCondition(doguResource *v3beta1.Dogu) bool {
	status := metav1.ConditionFalse
	reason := v3beta1.ReasonSchemaValidationEnabled
	message := "schema validation is enabled"
	if doguResource.Spec.SkipSchemaValidation {
		status = metav1.ConditionTrue
		reason = v3beta1.ReasonSchemaValidationDisabled
		message = "schema validation is skipped via spec.skipSchemaValidation"
	}

	return meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionSchemaValidationSkipped,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})
}

// abort sets ConditionValid to false with the given reason, persists changed conditions and returns an
// aborting StepResult so the use-case stops the reconciliation and emits a warning event.
func (vcs *ValidateChartStep) abort(ctx context.Context, doguResource *v3beta1.Dogu, skippedChanged bool, reason, message string) stepsv3.StepResult {
	validChanged := vcs.setValidCondition(doguResource, metav1.ConditionFalse, reason, message)
	if skippedChanged || validChanged {
		if err := vcs.persistConditions(ctx, doguResource); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
		}
	}

	return stepsv3.Abort(reason, message)
}

// succeed sets ConditionValid to true, persists changed conditions and continues the reconciliation.
func (vcs *ValidateChartStep) succeed(ctx context.Context, doguResource *v3beta1.Dogu, skippedChanged bool) stepsv3.StepResult {
	const message = "chart values satisfy the schema, the chart renders successfully and no volume shrinks or changes its storage class"
	validChanged := vcs.setValidCondition(doguResource, metav1.ConditionTrue, ReasonValid, message)
	if skippedChanged || validChanged {
		if err := vcs.persistConditions(ctx, doguResource); err != nil {
			return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
		}
		logging.Info(ctx, "Successfully validated dogu chart", "dogu", doguResource.Spec.Name)
	}

	return stepsv3.Continue()
}

func (vcs *ValidateChartStep) setValidCondition(doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) bool {
	return meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionValid,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})
}

func (vcs *ValidateChartStep) persistConditions(ctx context.Context, doguResource *v3beta1.Dogu) error {
	if err := vcs.k8sClient.Status().Update(ctx, doguResource); err != nil {
		logging.Error(ctx, err, "failed to update dogu validation conditions", "dogu", doguResource.Name)

		return fmt.Errorf("failed to update dogu validation conditions: %w", err)
	}

	return nil
}
