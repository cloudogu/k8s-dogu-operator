package controllers

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguChangeUseCase struct {
	steps []step
}

func NewDoguChangeUseCase(
	validationStep *ValidationStep,
	existsStep *FinalizerExistsStep,
	configStep *DoguConfigStep,
	doguReferenceStep *DoguConfigOwnerReferenceStep,
	sensitiveConfigStep *SensitiveConfigStep,
	sensitiveReferenceStep *SensitiveConfigOwnerReferenceStep,
	registerDoguVersionStep *RegisterDoguVersionStep,
	serviceAccountStep *ServiceAccountStep,
	volumeStep *VolumeStep,
	serviceStep *ServiceStep,
	customResourceStep *customK8sResourceStep,
	netPolsStep *networkPoliciesStep,
	deploymentStep *DeploymentStep,
	ingressAnnotationsStep *AdditionalIngressAnnotationsStep,
) *DoguChangeUseCase {
	return &DoguChangeUseCase{
		steps: []step{
			validationStep,
			existsStep,
			configStep,
			doguReferenceStep,
			sensitiveConfigStep,
			sensitiveReferenceStep,
			registerDoguVersionStep,
			serviceAccountStep,
			volumeStep,
			serviceStep,
			customResourceStep,
			netPolsStep,
			deploymentStep,
			ingressAnnotationsStep,
		},
	}
}

func (dcu *DoguChangeUseCase) HandleUntilApplied(ctx context.Context, doguResource *v2.Dogu) (time.Duration, error) {
	logger := log.FromContext(ctx).
		WithName("DoguChangeUseCase.HandleUntilApplied").
		WithValues("doguName", doguResource.Name)

	for _, s := range dcu.steps {
		requeueAfter, err := s.Run(ctx, doguResource)
		if err != nil || requeueAfter != 0 {
			logger.Error(err, "reconcile step has to requeue: %w", err)
			return requeueAfter, err
		}
	}
	return 0, nil
}
