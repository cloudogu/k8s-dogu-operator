package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	v2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2/helmfile"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const conditionHelmfileApplied = "HelmfileApplied"

type ExperimentalHelmfileStep struct {
	helmfileRuntime              *helmfile.Runtime
	isEnabled                    bool
	doguFetcher                  resourceDoguFetcher
	doguInterface                v2.DoguInterface
	localDoguFetcher             cesregistry.LocalDoguFetcher
	doguRegistrator              cesregistry.DoguRegistrator
	descriptorOwnerReferenceStep LocalDoguDescriptorOwnerReferenceStep
}

func NewExperimentalHelmfileStep(
	cfg *config.OperatorConfig,
	fetcher cesregistry.ResourceDoguFetcher,
	runtime *helmfile.Runtime,
	doguInterface v2.DoguInterface,
	localDoguFetcher cesregistry.LocalDoguFetcher,
	doguRegistrator cesregistry.DoguRegistrator,
	descriptorOwnerReferenceStep LocalDoguDescriptorOwnerReferenceStep,
) *ExperimentalHelmfileStep {
	return &ExperimentalHelmfileStep{
		helmfileRuntime:              runtime,
		isEnabled:                    cfg.ExperimentalHelmfileSupport,
		doguFetcher:                  fetcher,
		doguInterface:                doguInterface,
		localDoguFetcher:             localDoguFetcher,
		doguRegistrator:              doguRegistrator,
		descriptorOwnerReferenceStep: descriptorOwnerReferenceStep,
	}
}

func (e *ExperimentalHelmfileStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).
		WithName("experimentalHelmfileStep").
		WithValues("dogu", resource.Name)

	if !e.isEnabled {
		return steps.Continue()
	}

	doguDescriptor, _, err := e.doguFetcher.FetchWithResource(ctx, resource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguDescriptor.Properties["kind"] != "Helmfile" {
		logger.Info("dogu is not a helmfile dogu, continuing")
		return steps.Continue()
	}

	logger.Info("handling helmfile dogu")

	if resource.Name != "opendesk" {
		return steps.RequeueWithError(fmt.Errorf("openDesk is currently the only helmfile dogu supported"))
	}
	if !resource.DeletionTimestamp.IsZero() {
		return steps.RequeueWithError(fmt.Errorf("helmfile dogu is being deleted"))
	}
	if err := e.helmfileRuntime.UpdateFinalizer(ctx, resource, true); err != nil {
		return steps.RequeueWithError(err)
	}
	// register the dogu so it appears in the admin ui
	if err := e.registerDogu(ctx, resource, doguDescriptor); err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to register helmfile dogu: %w", err))
	}
	if result := e.descriptorOwnerReferenceStep.Run(ctx, resource); !result.Continue {
		return result
	}

	applied := meta.FindStatusCondition(resource.Status.Conditions, conditionHelmfileApplied)
	if applied == nil || applied.Status != metav1.ConditionTrue || applied.ObservedGeneration != resource.Generation {
		generation := resource.Generation
		updatedDogu, err := e.doguInterface.UpdateStatusWithRetry(ctx, resource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
			status.Status = doguv2.DoguStatusInstalling
			return status
		}, metav1.UpdateOptions{})
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to set helmfile dogu status to installing: %w", err))
		}
		*resource = *updatedDogu

		applyOptions, err := e.helmfileRuntime.Prepare(ctx, resource.Name, true)
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to extract and configure helmfile: %w", err))
		}

		if err := applyOptions.Run(ctx, "apply"); err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed to apply helmfile of %s: %w", resource.Name, err))
		}

		if err := e.setHelmfileApplied(ctx, resource, generation); err != nil {
			return steps.RequeueWithError(err)
		}
		if resource.Generation != generation {
			return steps.RequeueAfter(10 * time.Second)
		}
		logger.Info("Helmfile applied successfully")
	}

	if result := e.checkHelmfileDeploymentsReady(ctx, resource); !result.Continue {
		return result
	}

	return steps.Complete()
}

func (e *ExperimentalHelmfileStep) registerDogu(ctx context.Context, resource *doguv2.Dogu, descriptor *core.Dogu) error {
	enabled, err := e.localDoguFetcher.Enabled(ctx, resource.GetSimpleDoguName())
	if err != nil {
		return err
	}
	if !enabled {
		return e.doguRegistrator.RegisterNewDogu(ctx, resource, descriptor)
	}
	installed, err := e.localDoguFetcher.FetchInstalled(ctx, resource.GetSimpleDoguName())
	if err != nil {
		return err
	}
	if installed.Version == descriptor.Version {
		return nil
	}
	return e.doguRegistrator.RegisterDoguVersion(ctx, descriptor)
}

// checkHelmfileDeploymentsReady checks if all Helmfile Deployments are ready. If they aren't it requeues the resource.
func (e *ExperimentalHelmfileStep) checkHelmfileDeploymentsReady(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx)
	logger.Info("Checking readyness of dogu %s", resource.Name)
	namespace := e.helmfileRuntime.OpenDeskConfig.Namespace
	if namespace == "" {
		return steps.RequeueWithError(fmt.Errorf("deployment readiness check requires a namespace"))
	}

	var deployments appsv1.DeploymentList
	// use the global APIReader because Helmfile can deploy outside the namespace watched by the manager cache.
	if err := e.helmfileRuntime.APIReader.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to list helmfile deployments in namespace %s: %w", namespace, err))
	}

	// An empty list may mean that the resources are not visible yet.
	if len(deployments.Items) == 0 {
		return steps.RequeueAfter(10 * time.Second)
	}

	pendingDeployments := 0
	for _, deployment := range deployments.Items {
		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}

		status := deployment.Status
		if deployment.DeletionTimestamp != nil ||
			status.ObservedGeneration < deployment.Generation ||
			status.Replicas != desired ||
			status.UpdatedReplicas != desired ||
			status.ReadyReplicas != desired ||
			status.AvailableReplicas != desired {
			pendingDeployments++
		}
	}

	if pendingDeployments > 0 {
		logger.Info(fmt.Sprintf("Waiting for Helmfile deployments, %d/%d ready", len(deployments.Items)-pendingDeployments, len(deployments.Items)),
			"namespace", namespace)
		return steps.RequeueAfter(10 * time.Second)
	}

	updatedDogu, err := e.doguInterface.UpdateStatusWithRetry(ctx, resource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		status.Status = doguv2.DoguStatusInstalled
		status.InstalledVersion = resource.Spec.Version
		status.Health = doguv2.AvailableHealthStatus
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               doguv2.ConditionReady,
			Status:             metav1.ConditionTrue,
			Reason:             "HelmfileInstallationSucceeded",
			Message:            "Helmfile applied and all deployments are ready",
			ObservedGeneration: resource.Generation,
		})
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               doguv2.ConditionHealthy,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonDoguHealthy,
			Message:            "All replicas are available",
			ObservedGeneration: resource.Generation,
		})
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("update helmfile dogu readiness status: %w", err))
	}
	*resource = *updatedDogu
	return steps.Continue()
}

// setHelmfileApplied persists progress so readiness reconciliations and operator
// restarts do not repeat a successful apply for the same Dogu generation.
func (e *ExperimentalHelmfileStep) setHelmfileApplied(ctx context.Context, dogu *doguv2.Dogu, generation int64) error {
	updatedDoguResource, err := e.doguInterface.UpdateStatusWithRetry(ctx, dogu, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		condition := metav1.Condition{
			Type:               conditionHelmfileApplied,
			Status:             metav1.ConditionTrue,
			Reason:             "HelmfileApplySucceeded",
			Message:            "Helmfile applied successfully",
			ObservedGeneration: generation,
		}
		meta.SetStatusCondition(&status.Conditions, condition)
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update helmfile applied condition: %w", err)
	}
	*dogu = *updatedDoguResource
	return nil
}
