package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RevertStartupProbeStep struct {
	client              client.Client
	resourceDoguFetcher resourceDoguFetcher
	deploymentInterface deploymentInterface
	doguCommandExecutor exec.CommandExecutor
}

func NewRevertStartupProbeStep() *RevertStartupProbeStep {
	return &RevertStartupProbeStep{}
}

func (rsps *RevertStartupProbeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, _, err := rsps.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}
	deployment, err := rsps.deploymentInterface.Get(ctx, doguResource.Spec.Name, metav1.GetOptions{})
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch deployment: %w", err))
	}
	originalStartupProbe := resource.CreateStartupProbe(dogu)
	if rsps.startupProbeHasDefaultValue(deployment, dogu.GetSimpleName(), originalStartupProbe) {
		return steps.StepResult{}
	}

	fromDoguVersion := deployment.Annotations[previousDoguVersionAnnotationKey]

	// Run Postupgrade Script
	err = rsps.applyPostUpgradeScript(ctx, doguResource, fromDoguVersion, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("post-upgrade failed: %w", err))
	}

	// Revert probe
	err = rsps.revertStartupProbeAfterUpdate(ctx, doguResource, dogu, rsps.client)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	return steps.StepResult{}
}

func (rsps *RevertStartupProbeStep) startupProbeHasDefaultValue(deployment *v1.Deployment, containerName string, probe *coreV1.Probe) bool {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName && deployment.Spec.Template.Spec.Containers[i].StartupProbe != nil {
			return reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[i].StartupProbe, probe)
		}
	}
	return false
}

func (rsps *RevertStartupProbeStep) applyPostUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, toDogu *core.Dogu) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPostUpgrade) {
		return nil
	}

	postUpgradeCmd := toDogu.GetExposedCommand(core.ExposedCommandPostUpgrade)

	return rsps.executePostUpgradeScript(ctx, toDoguResource, fromDoguVersion, postUpgradeCmd)
}

func (rsps *RevertStartupProbeStep) executePostUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, postUpgradeCmd *core.ExposedCommand) error {
	postUpgradeShellCmd := exec.NewShellCommand(postUpgradeCmd.Command, fromDoguVersion, toDoguResource.Spec.Version)

	toDoguPod := &coreV1.Pod{}
	toDoguPod, getPodErr := toDoguResource.GetPod(ctx, rsps.client)
	if getPodErr != nil {
		return fmt.Errorf("failed to get new %s pod for post upgrade: %w", toDoguResource.Name, getPodErr)
	}

	outBuf, err := rsps.doguCommandExecutor.ExecCommandForPod(ctx, toDoguPod, postUpgradeShellCmd, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func (rsps *RevertStartupProbeStep) revertStartupProbeAfterUpdate(ctx context.Context, toDoguResource *v2.Dogu, toDogu *core.Dogu, client client.Client) error {
	originalStartupProbe := resource.CreateStartupProbe(toDogu)

	deployment, err := toDoguResource.GetDeployment(ctx, client)
	if err != nil {
		return err
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == toDoguResource.Name && container.StartupProbe != nil {
			deployment.Spec.Template.Spec.Containers[i].StartupProbe = originalStartupProbe
			return client.Update(ctx, deployment)
		}
	}

	return nil
}
