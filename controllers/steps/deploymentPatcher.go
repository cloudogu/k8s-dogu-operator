package steps

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	v2 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type DeploymentPatcher struct {
	deploymentInterface deploymentInterface
}

func NewDeploymentPatcher(deploymentInterface deploymentInterface) *DeploymentPatcher {
	return &DeploymentPatcher{
		deploymentInterface: deploymentInterface,
	}
}

func (dp *DeploymentPatcher) Execute(ctx context.Context, name string, patchData map[string]interface{}) (*v1.Deployment, error) {
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patch: %w", err)
	}
	return dp.deploymentInterface.Patch(ctx, name, types.MergePatchType, patchBytes, v2.PatchOptions{})
}
