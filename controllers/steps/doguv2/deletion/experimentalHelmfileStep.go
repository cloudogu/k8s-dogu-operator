package deletion

import (
	"context"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2/helmfile"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExperimentalHelmfileStep struct {
	helmfileRuntime *helmfile.Runtime
	isEnabled       bool
}

func NewExperimentalHelmfileStep(cfg *config.OperatorConfig, runtime *helmfile.Runtime) *ExperimentalHelmfileStep {
	return &ExperimentalHelmfileStep{helmfileRuntime: runtime, isEnabled: cfg.ExperimentalHelmfileSupport}
}

// Run uninstalls Helmfile releases before releasing the cleanup finalizer.
// The marker prevents normal Dogu cleanup if another finalizer keeps the resource alive.
func (e *ExperimentalHelmfileStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	if !controllerutil.ContainsFinalizer(resource, helmfile.Finalizer) {
		if resource.Annotations[helmfile.ManagedAnnotation] == "true" {
			if resource.DeletionTimestamp.IsZero() {
				return steps.RequeueWithError(fmt.Errorf("refusing to release cleanup finalizers before dogu deletion"))
			}
			return e.releaseCleanupFinalizers(ctx, resource)
		}
		return steps.Continue()
	}
	if resource.DeletionTimestamp.IsZero() {
		return steps.RequeueWithError(fmt.Errorf("refusing to destroy helmfile before dogu deletion"))
	}
	if !e.isEnabled {
		return steps.RequeueWithError(fmt.Errorf("helmfile support must be enabled to clean up the helmfile dogu"))
	}
	if resource.Name != "opendesk" {
		return steps.RequeueWithError(fmt.Errorf("openDesk is currently the only helmfile dogu supported"))
	}
	logger := log.FromContext(ctx).WithValues("dogu", resource.Name)
	logger.Info("Preparing Helmfile destruction")
	options, err := e.helmfileRuntime.Prepare(ctx, resource.Name, false)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("prepare helmfile destruction: %w", err))
	}
	if err := options.Run(ctx, "destroy"); err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to destroy helmfile of %s: %w", resource.Name, err))
	}
	return e.releaseCleanupFinalizers(ctx, resource)
}

func (e *ExperimentalHelmfileStep) releaseCleanupFinalizers(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	if !controllerutil.ContainsFinalizer(resource, helmfile.Finalizer) && !controllerutil.ContainsFinalizer(resource, finalizerName) {
		return steps.Complete()
	}
	if err := e.helmfileRuntime.UpdateFinalizer(ctx, resource, false, finalizerName); err != nil {
		return steps.RequeueWithError(err)
	}
	log.FromContext(ctx).Info("Helmfile cleanup completed", "dogu", resource.Name)
	return steps.Complete()
}
