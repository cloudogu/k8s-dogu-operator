package migration

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
)

type CleanupV2ResourcesStep struct {
	eventRecorder EventRecorder
}

func NewCleanupV2ResourcesStep(eventRecorder EventRecorder) *CleanupV2ResourcesStep {
	return &CleanupV2ResourcesStep{
		eventRecorder: eventRecorder,
	}
}

func (s *CleanupV2ResourcesStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	return stepsv3.Continue()
}
