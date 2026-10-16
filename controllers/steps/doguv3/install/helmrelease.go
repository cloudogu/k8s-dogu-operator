package install

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	flux "github.com/fluxcd/helm-controller/api/v2"
	fluxoci "github.com/fluxcd/source-controller/api/v1"
	core "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EnsureHelmReleaseStep ensures the HelmRelease resource.
type EnsureHelmReleaseStep struct {
	k8sClient             K8sClient
	eventRecorder         EventRecorder
	helmReconcileInterval metav1.Duration
	retryInterval         metav1.Duration
	chartService          ChartService
}

func NewEnsureHelmReleaseStep(k8sClient K8sClient, operatorConfig *config.OperatorConfig, chartService ChartService, recorder EventRecorder) *EnsureHelmReleaseStep {
	return &EnsureHelmReleaseStep{
		k8sClient:             k8sClient,
		eventRecorder:         recorder,
		helmReconcileInterval: metav1.Duration{Duration: operatorConfig.DoguHelmReconciliationInterval},
		retryInterval:         metav1.Duration{Duration: operatorConfig.DoguHelmRetryInterval},
		chartService:          chartService,
	}
}

func (ehr *EnsureHelmReleaseStep) Run(ctx context.Context, doguResource *doguv3.Dogu) stepsv3.StepResult {
	templateAsm := values3.NewAssembler(ehr.k8sClient)
	combinedValues, err := combineValues(ctx, doguResource, ehr.chartService, templateAsm)
	if err != nil {
		pending := meta.FindStatusCondition(doguResource.Status.Conditions, doguv3.ConditionUpdatePending)
		knownChange := pending != nil && pending.Status == metav1.ConditionTrue && pending.ObservedGeneration == doguResource.Generation
		if doguResource.Spec.Stopped && !knownChange {
			if statusErr := updatePending(ctx, ehr.k8sClient, doguResource, metav1.ConditionUnknown, err.Error()); statusErr != nil {
				return stepsv3.RequeueWithError(statusErr, doguv3.ReasonStopped)
			}
		}
		return stepsv3.RequeueWithError(err, doguv3.ReasonInstalling)
	}
	if doguResource.Spec.Stopped {
		return ehr.compareStoppedRelease(ctx, doguResource, combinedValues)
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
		log.FromContext(ctx).Info("created HelmRelease", "name", release.Name)
	case controllerutil.OperationResultUpdated:
		ehr.eventRecorder.Event(doguResource, core.EventTypeNormal, doguv3.ConditionChartAvailable, "HelmRelease updated")
		log.FromContext(ctx).Info("updated HelmRelease", "name", release.Name)
	case controllerutil.OperationResultNone:
		// nothing changed - HelmRelease already was exactly as desired
	default:
		// should not happen, as we never make changes to the HelmRelease's Status
		log.FromContext(ctx).Info("found unexpected operation result for helm release upsert", "result", result, "helm-release", release.Name)
	}

	if meta.FindStatusCondition(doguResource.Status.Conditions, doguv3.ConditionUpdatePending) != nil {
		if err := updatePending(ctx, ehr.k8sClient, doguResource, metav1.ConditionFalse, "The desired release has been handed to Flux"); err != nil {
			return stepsv3.RequeueWithError(err, doguv3.ReasonStarting)
		}
	}
	return stepsv3.Continue()
}

// compareStoppedRelease reports pending chart or values changes without updating the HelmRelease.
// The HelmRelease must remain suspended so Flux does not apply newly fetched chart versions.
func (ehr *EnsureHelmReleaseStep) compareStoppedRelease(ctx context.Context, dogu *doguv3.Dogu, values *apiext.JSON) stepsv3.StepResult {
	release := &flux.HelmRelease{}
	err := ehr.k8sClient.Get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		if err := updatePending(ctx, ehr.k8sClient, dogu, metav1.ConditionTrue, "Installation is pending until the dogu starts"); err != nil {
			return stepsv3.RequeueWithError(err, doguv3.ReasonStopped)
		}
		return stepsv3.Continue()
	}
	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonStopped)
	}

	desired := release.DeepCopy()
	configureHelmRelease(desired, dogu, values, &ehr.retryInterval, &ehr.helmReconcileInterval)
	desiredValues, liveValues := desired.GetValues(), release.GetValues()
	status, message := metav1.ConditionFalse, "No changes are pending"
	if release.Labels[doguv3.DoguLabelVersion] != dogu.Spec.Version ||
		!reflect.DeepEqual(desiredValues, liveValues) ||
		!reflect.DeepEqual(desired.Spec.ChartRef, release.Spec.ChartRef) ||
		!reflect.DeepEqual(desired.Spec.CommonMetadata, release.Spec.CommonMetadata) {
		status, message = metav1.ConditionTrue, "Chart or values changes are pending until the dogu starts"
	}
	if err := updatePending(ctx, ehr.k8sClient, dogu, status, message); err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonStopped)
	}
	return stepsv3.Continue()
}

// updatePending records whether changes are waiting to be passed to the HelmRelease.
// It writes the Dogu status only when the condition changes.
func updatePending(ctx context.Context, k8sClient client.Client, dogu *doguv3.Dogu, status metav1.ConditionStatus, message string) error {
	reason := doguv3.ReasonStopped
	if status == metav1.ConditionFalse {
		reason = doguv3.ReasonSucceeded
	}
	if meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type: doguv3.ConditionUpdatePending, Status: status, Reason: reason, Message: message, ObservedGeneration: dogu.Generation,
	}) {
		return k8sClient.Status().Update(ctx, dogu)
	}
	return nil
}

func configureHelmRelease(release *flux.HelmRelease, doguResource *doguv3.Dogu, values *apiext.JSON, retryInterval *metav1.Duration, helmReconcileInterval *metav1.Duration) {
	if release.Labels == nil {
		release.Labels = make(map[string]string)
	}
	release.Labels[labelKeyFluxSharding] = resource.LabelValueCes
	release.Labels[doguv3.DoguLabelName] = doguResource.Spec.Name
	release.Labels[doguv3.DoguLabelVersion] = doguResource.Spec.Version
	release.Spec = flux.HelmReleaseSpec{
		Suspend: release.Spec.Suspend,
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
		Values: values,
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

func combineValues(ctx context.Context, dogu *doguv3.Dogu, doguMetadataValueSvc ChartService, valueAssembler ValueAssembler) (*apiext.JSON, error) {
	metaValues, _, err := doguMetadataValueSvc.DoguMetaValues(ctx, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve dogu metadata values: %w", err)
	}

	values, err := valueAssembler.Assemble(ctx, dogu, metaValues)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble dogu values: %w", err)
	}

	// this case the error is probably unreachable because the assembler has already pushed the values object into a YAML parser.
	bytes, _ := json.Marshal(values)

	return &apiext.JSON{Raw: bytes}, nil
}
