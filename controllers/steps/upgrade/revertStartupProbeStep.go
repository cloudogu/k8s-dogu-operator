package upgrade

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterRevertStartupProbe = time.Second * 3

type RevertStartupProbeStep struct {
	client              k8sClient
	localDoguFetcher    localDoguFetcher
	deploymentInterface deploymentInterface
	doguCommandExecutor commandExecutor
}

func NewRevertStartupProbeStep(
	client client.Client,
	deploymentInterface appsv1.DeploymentInterface,
	localFetcher cesregistry.LocalDoguFetcher,
	executor exec.CommandExecutor,
) *RevertStartupProbeStep {
	return &RevertStartupProbeStep{
		client:              client,
		deploymentInterface: deploymentInterface,
		localDoguFetcher:    localFetcher,
		doguCommandExecutor: executor,
	}
}

func (rsps *RevertStartupProbeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	toDogu, err := rsps.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	deployment, err := rsps.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch deployment: %w", err))
	}

	originalStartupProbe := resource.CreateStartupProbe(toDogu)
	if rsps.startupProbeHasDefaultValue(deployment, toDogu.GetSimpleName(), originalStartupProbe) {
		return steps.Continue()
	}

	fromDogu, err := rsps.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch installed dogu: %w", err))
	}

	// Run Postupgrade Script
	err = rsps.applyPostUpgradeScript(ctx, doguResource, fromDogu.Version, toDogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("post-upgrade failed: %w", err))
	}

	// Revert probe
	err = rsps.revertStartupProbeAfterUpdate(ctx, doguResource, toDogu, deployment)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.RequeueAfter(requeueAfterRevertStartupProbe)
}

func (rsps *RevertStartupProbeStep) startupProbeHasDefaultValue(deployment *v1.Deployment, containerName string, probe *coreV1.Probe) bool {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
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

	outBuf, err := rsps.doguCommandExecutor.ExecCommandForPod(ctx, toDoguPod, postUpgradeShellCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func (rsps *RevertStartupProbeStep) revertStartupProbeAfterUpdate(ctx context.Context, toDoguResource *v2.Dogu, toDogu *core.Dogu, deployment *v1.Deployment) error {
	originalStartupProbe := resource.CreateStartupProbe(toDogu)

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == toDoguResource.Name && container.StartupProbe != nil {
			deployment.Spec.Template.Spec.Containers[i].StartupProbe = originalStartupProbe
			_, err := rsps.deploymentInterface.Update(ctx, deployment, metav1.UpdateOptions{})
			return err
		}
	}

	return nil
}
