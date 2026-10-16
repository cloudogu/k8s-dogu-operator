package doguv3

import (
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
)

// StepFactory selects the step chain for the desired Dogu state.
type StepFactory struct {
	runningSteps  []Step
	stoppedSteps  []Step
	deletionSteps []Step
}

func NewStepFactory(
	finalizerStep *install.CreateFinalizerStep,
	stopStep *install.StopWorkloadsStep,
	ociStep *install.EnsureOCIRepositoryStep,
	waitOCIStep *install.WaitForOCIRepositoryReadyStep,
	validateChartStep *install.ValidateChartStep,
	helmReleaseStep *install.EnsureHelmReleaseStep,
	helmReleaseStatusStep *install.HelmReleaseStatusStep,
) StepFactory {
	return StepFactory{
		// Both chains set the finalizer before creating owned resources.
		runningSteps: []Step{finalizerStep, stopStep, ociStep, waitOCIStep, validateChartStep, helmReleaseStep, helmReleaseStatusStep},
		stoppedSteps: []Step{finalizerStep, stopStep, ociStep, waitOCIStep, helmReleaseStep, validateChartStep},
	}
}

func (f *StepFactory) StepsFor(dogu *v3beta1.Dogu) []Step {
	if !dogu.GetDeletionTimestamp().IsZero() {
		return f.deletionSteps
	}
	if dogu.Spec.Stopped {
		return f.stoppedSteps
	}
	return f.runningSteps
}
