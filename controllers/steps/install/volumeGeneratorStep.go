package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VolumeGeneratorStep struct {
	localDoguFetcher    localDoguFetcher
	deploymentPatcher   steps.DeploymentPatcher
	deploymentInterface deploymentInterface
}

func NewVolumeGeneratorStep(mgrSet *util.ManagerSet, deploymentPatcher steps.DeploymentPatcher, deploymentInterface deploymentInterface) *VolumeGeneratorStep {
	return &VolumeGeneratorStep{
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		deploymentPatcher:   deploymentPatcher,
		deploymentInterface: deploymentInterface,
	}
}

func (vgs *VolumeGeneratorStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := vgs.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err))
	}

	volumes, err := resource.CreateVolumes(doguResource, dogu, doguResource.Spec.ExportMode)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}

	deployment, err := vgs.getDoguDeployment(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"volumes": volumes,
				},
			},
		},
	}

	_, err = vgs.deploymentPatcher.Execute(ctx, deployment.Name, patch)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	return steps.StepResult{}
}

func (vgs *VolumeGeneratorStep) getDoguDeployment(ctx context.Context, doguResource *v2.Dogu) (*appsv1.Deployment, error) {
	list, err := vgs.deploymentInterface.List(ctx, v1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v2.DoguLabelName, doguResource.GetObjectKey().Name)})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment for dogu %s: %w", doguResource.Name, err)
	}
	if len(list.Items) == 1 {
		return &list.Items[0], nil
	}

	return nil, fmt.Errorf("dogu %s has more than one or zero deployments", doguResource.GetObjectKey().Name)
}
