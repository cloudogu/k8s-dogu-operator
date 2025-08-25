package upgrade

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const podTemplateVersionKey = "dogu.version"
const upgradeStartupProbeFailureThresholdRetries = int32(1080)
const preUpgradeScriptDir = "/tmp/pre-upgrade"

type UpdateDeploymentStep struct {
	client              client.Client
	upserter            resource.ResourceUpserter
	deploymentInterface deploymentInterface
	resourceDoguFetcher resourceDoguFetcher
	execPodFactory      exec.ExecPodFactory
	doguCommandExecutor exec.CommandExecutor
}

func NewUpdateDeploymentStep() *UpdateDeploymentStep {
	return &UpdateDeploymentStep{}
}

func (uds *UpdateDeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := uds.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	updated := uds.isDeploymentStartupProbeIncreased(doguResource, deployment) && uds.isDoguVersionUpdatedInDeployment(doguResource, deployment)
	if updated {
		return steps.StepResult{}
	}
	dogu, _, err := uds.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}
	fromDoguVersion := deployment.Spec.Template.Labels[podTemplateVersionKey]

	// Start exec pod
	execPod, err := uds.execPodFactory.NewExecPod(doguResource, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	err = execPod.Create(ctx)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	defer deleteExecPod(ctx, execPod)

	// Apply pre upgrade
	err = uds.applyPreUpgradeScript(ctx, doguResource, fromDoguVersion, dogu, execPod)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("pre-upgrade failed: %w", err))
	}

	// update Deployment
	_, err = uds.upserter.UpsertDoguDeployment(
		ctx,
		doguResource,
		dogu,
		func(deployment *v1.Deployment) {
			increaseStartupProbeTimeoutForUpdate(doguResource.Name, deployment)
		},
	)

	return steps.StepResult{}
}

func (uds *UpdateDeploymentStep) isDeploymentStartupProbeIncreased(doguResource *v2.Dogu, deployment *v1.Deployment) bool {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		startupProbe := deployment.Spec.Template.Spec.Containers[i].StartupProbe
		if container.Name == doguResource.Name && startupProbe != nil && startupProbe.FailureThreshold == upgradeStartupProbeFailureThresholdRetries {
			return true
		}
	}
	return false
}

func (uds *UpdateDeploymentStep) isDoguVersionUpdatedInDeployment(doguResource *v2.Dogu, deployment *v1.Deployment) bool {
	return deployment.Spec.Template.Labels[podTemplateVersionKey] == doguResource.Spec.Version
}

func increaseStartupProbeTimeoutForUpdate(containerName string, deployment *v1.Deployment) {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName && deployment.Spec.Template.Spec.Containers[i].StartupProbe != nil {
			deployment.Spec.Template.Spec.Containers[i].StartupProbe.FailureThreshold = upgradeStartupProbeFailureThresholdRetries
			break
		}
	}
}

func deleteExecPod(ctx context.Context, execPod exec.ExecPod) {
	err := execPod.Delete(ctx)
	if err != nil {
		return
	}
}

func (uds *UpdateDeploymentStep) applyPreUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, toDogu *core.Dogu, execPod exec.ExecPod) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPreUpgrade) {
		return nil
	}

	preUpgradeScriptCmd := toDogu.GetExposedCommand(core.ExposedCommandPreUpgrade)

	fromDoguPod, err := getPodForDogu(ctx, uds.client, toDogu.GetSimpleName(), fromDoguVersion)
	if err != nil {
		return fmt.Errorf("failed to find pod for dogu %s:%s : %w", toDogu.GetSimpleName(), fromDoguVersion, err)
	}

	err = uds.copyPreUpgradeScriptFromPodToPod(ctx, execPod, fromDoguPod, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	err = uds.applyPreUpgradeScriptToOlderDogu(ctx, fromDoguVersion, fromDoguPod, toDoguResource, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	return nil
}

func (uds *UpdateDeploymentStep) copyPreUpgradeScriptFromPodToPod(ctx context.Context, srcPod exec.ExecPod, destPod *corev1.Pod, preUpgradeScriptCmd *core.ExposedCommand) error {
	tarCommand := exec.NewShellCommand("/bin/tar", "cf", "-", preUpgradeScriptCmd.Command)
	archive, err := srcPod.Exec(ctx, tarCommand)
	if err != nil {
		return fmt.Errorf("failed to get pre-upgrade script from execpod with command '%s', stdout: '%s':  %w", tarCommand.String(), archive, err)
	}

	createPathCommand := exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)
	out, err := uds.doguCommandExecutor.ExecCommandForPod(ctx, destPod, createPathCommand, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to create pre-upgrade target dir with command '%s', stdout: '%s': %w", createPathCommand.String(), out, err)
	}

	untarCommand := exec.NewShellCommandWithStdin(archive, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)
	out, err = uds.doguCommandExecutor.ExecCommandForPod(ctx, destPod, untarCommand, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to extract pre-upgrade script to dogu pod with command '%s', stdout: '%s': %w", untarCommand.String(), out, err)
	}

	return nil
}

func (uds *UpdateDeploymentStep) applyPreUpgradeScriptToOlderDogu(
	ctx context.Context,
	fromDoguVersion string,
	fromDoguPod *corev1.Pod,
	toDoguResource *v2.Dogu,
	preUpgradeCmd *core.ExposedCommand,
) error {
	logger := log.FromContext(ctx)
	logger.Info("applying pre-upgrade script to old dogu")

	preUpgradeScriptPath := filepath.Join(preUpgradeScriptDir, filepath.Base(preUpgradeCmd.Command))

	preUpgradeShellCmd := exec.NewShellCommand(preUpgradeScriptPath, fromDoguVersion, toDoguResource.Spec.Version)

	logger.Info("Executing pre-upgrade command " + preUpgradeShellCmd.String())
	outBuf, err := uds.doguCommandExecutor.ExecCommandForPod(ctx, fromDoguPod, preUpgradeShellCmd, exec.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func getPodForDogu(ctx context.Context, cli client.Client, fromDoguName string, fromDoguVersion string) (*corev1.Pod, error) {
	fromDoguLabels := map[string]string{
		v2.DoguLabelName:    fromDoguName,
		v2.DoguLabelVersion: fromDoguVersion,
	}

	return v2.GetPodForLabels(ctx, cli, fromDoguLabels)
}
