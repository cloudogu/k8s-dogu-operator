package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	flux "github.com/fluxcd/helm-controller/api/v2"
	fluxoci "github.com/fluxcd/source-controller/api/v1"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EnsureHelmReleaseStep ensures the HelmRelease resource.
type EnsureHelmReleaseStep struct {
	k8sClient             K8sClient
	eventRecorder         EventRecorder
	helmReconcileInterval metav1.Duration
	retryInterval         metav1.Duration
	chartService          chartService
	assembler             valueAssembler
}

func NewEnsureHelmReleaseStep(k8sClient K8sClient, operatorConfig *config.OperatorConfig, chartService chartService, assembler valueAssembler, recorder EventRecorder) *EnsureHelmReleaseStep {
	return &EnsureHelmReleaseStep{
		k8sClient:             k8sClient,
		eventRecorder:         recorder,
		helmReconcileInterval: metav1.Duration{Duration: operatorConfig.DoguHelmReconciliationInterval},
		retryInterval:         metav1.Duration{Duration: operatorConfig.DoguHelmRetryInterval},
		chartService:          chartService,
		assembler:             assembler,
	}
}

func (ehr *EnsureHelmReleaseStep) Run(ctx context.Context, doguResource *doguv3.Dogu) stepsv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		return stepsv3.Continue()
	}
	combinedValues, err := combineValues(ctx, doguResource, ehr.chartService, ehr.assembler)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to prepare HelmRelease values for dogu %s/%s: %w", doguResource.Namespace, doguResource.Name, err), doguv3.ReasonInstalling)
	}

	release := &flux.HelmRelease{
		Namespace: doguResource.Namespace,
		Name:      doguResource.Spec.Name,
	}

	// Use patch because the repository resource will be updated by the helm-controller.
	// CreateOrUpdate would produce conflict errors and increase the number of reconciles.
	result, err := controllerutil.CreateOrPatch(ctx, ehr.k8sClient, release, func() error {
		configureHelmRelease(release, doguResource, combinedValues, &ehr.retryInterval, &ehr.helmReconcileInterval)
		return controllerutil.SetControllerReference(doguResource, release, ehr.k8sClient.Scheme())
	})

	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to createOrPatch HelmRelease %q: %w", doguResource.Spec.Name, err), doguv3.ReasonInstalling)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		ehr.eventRecorder.Event(doguResource, core.EventTypeNormal, doguv3.ConditionChartAvailable, "HelmRelease created")
		logging.Info(ctx, "created HelmRelease", "name", release.Name)
	case controllerutil.OperationResultUpdated:
		ehr.eventRecorder.Event(doguResource, core.EventTypeNormal, doguv3.ConditionChartAvailable, "HelmRelease updated")
		logging.Info(ctx, "updated HelmRelease", "name", release.Name)
	case controllerutil.OperationResultNone:
		// nothing changed - HelmRelease already was exactly as desired
	default:
		// should not happen, as we never make changes to the HelmRelease's Status
		logging.Info(ctx, "found unexpected operation result for helm release upsert", "result", result, "helm-release", release.Name)
	}

	return stepsv3.Continue()
}

func configureHelmRelease(release *flux.HelmRelease, doguResource *doguv3.Dogu, values *apiext.JSON, retryInterval *metav1.Duration, helmReconcileInterval *metav1.Duration) {
	if release.Labels == nil {
		release.Labels = make(map[string]string)
	}
	release.Labels[labelKeyFluxSharding] = resource.LabelValueCes
	release.Labels[doguv3.DoguLabelName] = doguResource.Spec.Name
	release.Labels[doguv3.DoguLabelVersion] = doguResource.Spec.Version
	release.Spec = flux.HelmReleaseSpec{
		ChartRef: &flux.CrossNamespaceSourceReference{
			APIVersion: "v1",
			Kind:       fluxoci.OCIRepositoryKind,
			Namespace:  doguResource.Namespace,
			Name:       doguResource.Name,
		},
		Install: &flux.Install{
			DisableSchemaValidation: doguResource.Spec.SkipSchemaValidation,
			Strategy: &flux.InstallStrategy{
				Name:          string(flux.ActionStrategyRetryOnFailure),
				RetryInterval: retryInterval,
			},
		},
		Upgrade: &flux.Upgrade{
			DisableSchemaValidation: doguResource.Spec.SkipSchemaValidation,
			Strategy: &flux.UpgradeStrategy{
				Name:          string(flux.ActionStrategyRetryOnFailure),
				RetryInterval: retryInterval,
			},
		},
		Interval: *helmReconcileInterval,
		DriftDetection: &flux.DriftDetection{
			Mode:   flux.DriftDetectionEnabled,
			Ignore: []flux.IgnoreRule{},
		},
		// PVC resize, as the third possible reason for a suspension, is handled in earlier steps
		Suspend: doguResource.Spec.Stopped || doguResource.Spec.PauseReconciliation,
		Values:  values,
		CommonMetadata: &flux.CommonMetadata{
			Annotations: nil,
			Labels: map[string]string{
				resource.LegacyLabelKeyApp:     resource.LabelValueCes,
				resource.CloudoguLabelKeyApp:   resource.LabelValueCes,
				doguv2.DoguLabelName:           doguResource.Spec.Name,
				doguv3.DoguLabelName:           doguResource.Spec.Name,
				resource.CommonLabelKeyName:    doguResource.Spec.Name,
				resource.CommonLabelKeyVersion: doguResource.Spec.Version,
				resource.CommonLabelKeyPartOf:  resource.LabelValueCes,
			},
		},
	}
}

func combineValues(ctx context.Context, dogu *doguv3.Dogu, chartService chartService, assembler valueAssembler) (*apiext.JSON, error) {
	values, err := assembleValues(ctx, dogu, chartService, assembler)
	if err != nil {
		return nil, err
	}

	valueBytes, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize dogu values: %w", err)
	}

	return &apiext.JSON{Raw: valueBytes}, nil
}

func assembleValues(ctx context.Context, dogu *doguv3.Dogu, chartService chartService, assembler valueAssembler) (values3.Values, error) {
	valuesMeta, _, err := chartService.DoguMetaValues(ctx, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dogu metadata values: %w", err)
	}

	runtimePatchTemplate, found, err := chartService.RuntimePatchTemplate(ctx, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve runtime patch template: %w", err)
	}

	isEmpty := len(bytes.TrimSpace(runtimePatchTemplate)) == 0
	emptyPatchFileExists := found && isEmpty
	if emptyPatchFileExists {
		return nil, errors.New(values3.RuntimePatchTemplateFileName + " must not be empty")
	}

	values, err := assembler.Assemble(ctx, dogu, valuesMeta, runtimePatchTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble dogu values: %w", err)
	}

	return values, nil
}
