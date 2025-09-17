package upgrade

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const requeueAfterUpdateDeployment = time.Second * 3
const podTemplateVersionKey = "dogu.version"
const upgradeStartupProbeFailureThresholdRetries = int32(1080)
const preUpgradeScriptDir = "/tmp/pre-upgrade"
const previousDoguVersionAnnotationKey = "k8s.cloudogu.com/dogu-previous-version"

type UpdateDeploymentStep struct {
	client              k8sClient
	upserter            resourceUpserter
	deploymentInterface deploymentInterface
	resourceDoguFetcher resourceDoguFetcher
	localDoguFetcher    localDoguFetcher
	execPodFactory      execPodFactory
	doguCommandExecutor commandExecutor
}

func NewUpdateDeploymentStep(
	client client.Client,
	upserter resource.ResourceUpserter,
	deploymentInterface appsv1.DeploymentInterface,
	localFetcher cesregistry.LocalDoguFetcher,
	resourceFetcher cesregistry.ResourceDoguFetcher,
	factory exec.ExecPodFactory,
	executor exec.CommandExecutor,
) *UpdateDeploymentStep {
	return &UpdateDeploymentStep{
		client:              client,
		upserter:            upserter,
		deploymentInterface: deploymentInterface,
		localDoguFetcher:    localFetcher,
		resourceDoguFetcher: resourceFetcher,
		execPodFactory:      factory,
		doguCommandExecutor: executor,
	}
}

func (uds *UpdateDeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := uds.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if uds.isDoguVersionUpdatedInDeployment(doguResource, deployment) {
		return steps.Continue()
	}

	dogu, _, err := uds.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	execPodExists := uds.execPodFactory.Exists(ctx, doguResource, dogu)
	if !execPodExists {
		return steps.RequeueAfter(requeueAfterUpdateDeployment)
	}

	err = uds.execPodFactory.CheckReady(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to check if exec pod is ready: %w", err))
	}

	fromDogu, err := uds.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	// Apply pre upgrade
	err = uds.applyPreUpgradeScript(ctx, doguResource, fromDogu.Version, dogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("pre-upgrade failed: %w", err))
	}

	// update Deployment
	_, err = uds.upserter.UpsertDoguDeployment(
		ctx,
		doguResource,
		dogu,
		func(deployment *v1.Deployment) {
			increaseStartupProbeTimeoutForUpdate(doguResource.Name, deployment)
			util.SetPreviousDoguVersionInAnnotations(dogu.Version, deployment)
		},
	)

	return steps.RequeueAfter(requeueAfterUpdateDeployment)
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

func (uds *UpdateDeploymentStep) applyPreUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, toDogu *core.Dogu) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPreUpgrade) {
		return nil
	}

	preUpgradeScriptCmd := toDogu.GetExposedCommand(core.ExposedCommandPreUpgrade)

	fromDoguPod, err := toDoguResource.GetPod(ctx, uds.client)
	if err != nil {
		return fmt.Errorf("failed to find pod for dogu %s:%s : %w", toDogu.GetSimpleName(), fromDoguVersion, err)
	}

	err = uds.copyPreUpgradeScriptFromPodToPod(ctx, toDoguResource, toDogu, fromDoguPod, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	err = uds.applyPreUpgradeScriptToOlderDogu(ctx, fromDoguVersion, fromDoguPod, toDoguResource, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	return nil
}

func (uds *UpdateDeploymentStep) copyPreUpgradeScriptFromPodToPod(ctx context.Context, toDoguResource *v2.Dogu, toDogu *core.Dogu, destPod *corev1.Pod, preUpgradeScriptCmd *core.ExposedCommand) error {
	tarCommand := exec.NewShellCommand("/bin/tar", "cf", "-", preUpgradeScriptCmd.Command)
	archive, err := uds.execPodFactory.Exec(ctx, toDoguResource, toDogu, tarCommand)
	if err != nil {
		return fmt.Errorf("failed to get pre-upgrade script from execpod with command '%s', stdout: '%s':  %w", tarCommand.String(), archive, err)
	}

	createPathCommand := exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)
	out, err := uds.doguCommandExecutor.ExecCommandForPod(ctx, destPod, createPathCommand)
	if err != nil {
		return fmt.Errorf("failed to create pre-upgrade target dir with command '%s', stdout: '%s': %w", createPathCommand.String(), out, err)
	}

	untarCommand := exec.NewShellCommandWithStdin(archive, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)
	out, err = uds.doguCommandExecutor.ExecCommandForPod(ctx, destPod, untarCommand)
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
	outBuf, err := uds.doguCommandExecutor.ExecCommandForPod(ctx, fromDoguPod, preUpgradeShellCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeShellCmd, outBuf, err)
	}

	return nil
}
