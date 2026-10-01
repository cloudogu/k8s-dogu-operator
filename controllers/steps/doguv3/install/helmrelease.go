package install

import (
	"context"
	"encoding/json"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	flux "github.com/fluxcd/helm-controller/api/v2"
	fluxoci "github.com/fluxcd/source-controller/api/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EnsureHelmReleaseStep ensures the HelmRelease resource.
type EnsureHelmReleaseStep struct {
	k8sClient             K8sClient
	eventRecorder         EventRecorder
	helmReconcileInterval metav1.Duration
	retryInterval         metav1.Duration
	globalConfigRepo      resource.GlobalConfigRepository
}

func NewEnsureHelmReleaseStep(k8sClient K8sClient, operatorConfig config.OperatorConfig, globalConfigRepo resource.GlobalConfigRepository, recorder EventRecorder) *EnsureHelmReleaseStep {
	return &EnsureHelmReleaseStep{
		k8sClient:             k8sClient,
		eventRecorder:         recorder,
		helmReconcileInterval: metav1.Duration{Duration: operatorConfig.HelmReconciliationInterval},
		retryInterval:         metav1.Duration{Duration: operatorConfig.HelmRetryInterval},
		globalConfigRepo:      globalConfigRepo,
	}
}

func (ehr *EnsureHelmReleaseStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	//globalConfig, err := ehr.globalConfigRepo.Get(ctx)
	//if err != nil {
	//	err = fmt.Errorf("failed to get global config to configure dogu %s:%s: %w", doguResource.Namespace, doguResource.Name, err)
	//	return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	//}

	templateAsm := values3.NewAssembler(ehr.k8sClient)
	values, err := combineValues(ctx, doguResource, templateAsm)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonInstalling)
	}

	release := &flux.HelmRelease{
		Namespace: doguResource.Namespace,
		Name:      doguResource.Spec.Name,
	}

	// Use patch because the repository resource will be updated by the helm-controller.
	// CreateOrUpdate would produce conflict errors and increase the number of reconciles.
	result, err := controllerutil.CreateOrPatch(ctx, ehr.k8sClient, release, func() error {
		if release.Labels == nil {
			release.Labels = make(map[string]string)
		}
		release.Labels[labelKeyFluxSharding] = resource.LabelValueCes
		release.Labels[v3beta1.DoguLabelName] = doguResource.Spec.Name
		release.Labels[v3beta1.DoguLabelVersion] = doguResource.Spec.Version
		release.Spec = flux.HelmReleaseSpec{
			ChartRef: &flux.CrossNamespaceSourceReference{
				APIVersion: "v1",
				Kind:       fluxoci.OCIRepositoryKind,
				Namespace:  doguResource.Namespace,
				Name:       doguResource.Name,
			},
			Install: &flux.Install{
				Strategy: &flux.InstallStrategy{
					Name:          string(flux.ActionStrategyRetryOnFailure),
					RetryInterval: &ehr.retryInterval,
				},
			},
			Upgrade: &flux.Upgrade{
				Strategy: &flux.UpgradeStrategy{
					Name:          string(flux.ActionStrategyRetryOnFailure),
					RetryInterval: &ehr.retryInterval,
				},
			},
			Interval: ehr.helmReconcileInterval,
			DriftDetection: &flux.DriftDetection{
				Mode:   flux.DriftDetectionEnabled,
				Ignore: []flux.IgnoreRule{},
			},
			Values: values,
			CommonMetadata: &flux.CommonMetadata{
				Annotations: nil,
				Labels: map[string]string{
					resource.LabelKeyApp:                    resource.LabelValueCes,
					resource.LabelKeyK8sCloudoguComApp:      resource.LabelValueCes,
					v2.DoguLabelName:                        doguResource.Spec.Name,
					v3beta1.DoguLabelName:                   doguResource.Spec.Name,
					resource.LabelKeyAppKubernetesIoName:    doguResource.Spec.Name,
					resource.LabelKeyAppKubernetesIoVersion: doguResource.Spec.Version,
					resource.LabelKeyAppKubernetesIoPartOf:  resource.LabelValueCes,
				},
			},
		}

		return controllerutil.SetControllerReference(doguResource, release, ehr.k8sClient.Scheme())
	})

	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to createOrPatch HelmRelease %q: %w", doguResource.Spec.Name, err), v3beta1.ReasonInstalling)
	}

	switch result {
	case controllerutil.OperationResultCreated:
		ehr.eventRecorder.Event(doguResource, core.EventTypeNormal, v3beta1.ConditionChartAvailable, "HelmRelease created")
		log.FromContext(ctx).Info("created HelmRelease", "name", release.Name)
	case controllerutil.OperationResultUpdated:
		ehr.eventRecorder.Event(doguResource, core.EventTypeNormal, v3beta1.ConditionChartAvailable, "HelmRelease updated")
		log.FromContext(ctx).Info("updated HelmRelease", "name", release.Name)
	default:
		log.FromContext(ctx).Info("found unexpected operation result for helm release upsert", "result", result, "helm-release", release.Name)
	}

	return stepsv3.Continue()
}

func combineValues(ctx context.Context, dogu *v3beta1.Dogu, assembler valueAssembler) (*v1.JSON, error) {
	var noExtraPatchTpl []byte
	values, err := assembler.Assemble(ctx, dogu, noExtraPatchTpl)

	if err != nil {
		return nil, err
	}
	bytes, err := json.Marshal(values)

	if err != nil {
		return nil, err
	}

	return &v1.JSON{Raw: bytes}, nil
}
