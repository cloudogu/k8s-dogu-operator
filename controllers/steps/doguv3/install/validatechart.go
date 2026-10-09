package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
// cluster resource. A chart whose values violate the schema or that fails to render aborts the
// reconciliation via ConditionValid so no HelmRelease is produced.
type ValidateChartStep struct {
	chartService ChartService
	assembler    ValueAssembler
	k8sClient    K8sClient
}

func NewValidateChartStep(chartService ChartService, assembler ValueAssembler, k8sClient K8sClient) *ValidateChartStep {
	return &ValidateChartStep{
		chartService: chartService,
		assembler:    assembler,
		k8sClient:    k8sClient,
	}
}

func (vcs *ValidateChartStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	logger := log.FromContext(ctx)

	metaValues, _, err := vcs.chartService.DoguMetaValues(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get dogu meta values: %w", err), v3beta1.ReasonInstalling)
	}

	patchTemplate, found, err := vcs.chartService.ChartPatchTemplate(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to retrieve chart patch template: %w", err), v3beta1.ReasonInstalling)
	}
	if !found {
		patchTemplate = nil
	} else if patchTemplate == nil {
		patchTemplate = []byte{}
	}

	values, err := vcs.assembler.Assemble(ctx, doguResource, metaValues, patchTemplate)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to assemble values: %w", err), v3beta1.ReasonInstalling)
	}

	// Record whether schema validation is skipped and, unless skipped, validate the values.
	vcs.setSchemaValidationSkippedCondition(doguResource)
	if !doguResource.Spec.SkipSchemaValidation {
		if vErr := vcs.chartService.ValidateValues(ctx, doguResource, values); vErr != nil {
			msg := fmt.Sprintf("values do not satisfy the chart schema: %v", vErr)

			return vcs.abort(ctx, doguResource, v3beta1.ReasonSchemaInvalid, msg)
		}

		logger.Info("Successfully validated values against schema for dogu", "dogu", doguResource.Spec.Name)
	}

	if _, rErr := vcs.chartService.Render(ctx, doguResource, values); rErr != nil {
		msg := fmt.Sprintf("chart could not be rendered: %v", rErr)
		return vcs.abort(ctx, doguResource, ReasonRenderFailed, msg)
	}

	logger.Info("Successfully rendered dogu helm chart", "dogu", doguResource.Spec.Name)

	return vcs.succeed(ctx, doguResource)
}

// setSchemaValidationSkippedCondition mirrors the Spec.SkipSchemaValidation flag onto the
// ConditionSchemaValidationSkipped condition. It only mutates the in-memory resource; the change is
// persisted together with ConditionValid when the step reaches an exit point.
func (vcs *ValidateChartStep) setSchemaValidationSkippedCondition(doguResource *v3beta1.Dogu) {
	status := metav1.ConditionFalse
	reason := v3beta1.ReasonSchemaValidationEnabled
	message := "schema validation is enabled"
	if doguResource.Spec.SkipSchemaValidation {
		status = metav1.ConditionTrue
		reason = v3beta1.ReasonSchemaValidationDisabled
		message = "schema validation is skipped via spec.skipSchemaValidation"
	}

	meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionSchemaValidationSkipped,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})
}

// abort sets ConditionValid to false with the given reason, persists the conditions and returns an
// aborting StepResult so the use-case stops the reconciliation and emits a warning event.
func (vcs *ValidateChartStep) abort(ctx context.Context, doguResource *v3beta1.Dogu, reason, message string) stepsv3.StepResult {
	vcs.setValidCondition(doguResource, metav1.ConditionFalse, reason, message)
	if err := vcs.persistConditions(ctx, doguResource); err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	return stepsv3.Abort(reason, message)
}

// succeed sets ConditionValid to true, persists the conditions and continues the reconciliation.
func (vcs *ValidateChartStep) succeed(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	const message = "chart values satisfy the schema and the chart renders successfully"
	vcs.setValidCondition(doguResource, metav1.ConditionTrue, ReasonValid, message)
	if err := vcs.persistConditions(ctx, doguResource); err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	return stepsv3.Continue()
}

func (vcs *ValidateChartStep) setValidCondition(doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionValid,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})
}

func (vcs *ValidateChartStep) persistConditions(ctx context.Context, doguResource *v3beta1.Dogu) error {
	if err := vcs.k8sClient.Status().Update(ctx, doguResource); err != nil {
		log.FromContext(ctx).Error(err, "failed to update dogu validation conditions", "dogu", doguResource.Name)

		return fmt.Errorf("failed to update dogu validation conditions: %w", err)
	}

	return nil
}
