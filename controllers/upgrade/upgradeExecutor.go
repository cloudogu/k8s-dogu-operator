package upgrade

import (
	"context"
	"fmt"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

const (
	EventReason                     = "Upgrading"
	ErrorOnFailedUpgradeEventReason = "ErrUpgrade"
)

// upgradeStartupProbeFailureThresholdRetries contains the number of times how often a startup probe may fail. This
// value will be multiplied with 10 seconds for each timeout so that f. i. 1080 timeouts lead to a threshold of 3 hours.
const upgradeStartupProbeFailureThresholdRetries = int32(1080)

const preUpgradeScriptDir = "/tmp/pre-upgrade"

type upgradeExecutor struct {
	client                client.Client
	eventRecorder         record.EventRecorder
	imageRegistry         cloudogu.ImageRegistry
	collectApplier        cloudogu.CollectApplier
	k8sFileExtractor      cloudogu.FileExtractor
	serviceAccountCreator cloudogu.ServiceAccountCreator
	doguRegistrator       cloudogu.DoguRegistrator
	resourceUpserter      cloudogu.ResourceUpserter
	execPodFactory        cloudogu.ExecPodFactory
	doguCommandExecutor   cloudogu.CommandExecutor
}

// NewUpgradeExecutor creates a new upgrade executor.
func NewUpgradeExecutor(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *upgradeExecutor {
	return &upgradeExecutor{
		client:                client,
		eventRecorder:         eventRecorder,
		imageRegistry:         mgrSet.ImageRegistry,
		collectApplier:        mgrSet.CollectApplier,
		k8sFileExtractor:      mgrSet.FileExtractor,
		serviceAccountCreator: mgrSet.ServiceAccountCreator,
		doguRegistrator:       mgrSet.DoguRegistrator,
		resourceUpserter:      mgrSet.ResourceUpserter,
		execPodFactory:        exec.NewExecPodFactory(client, mgrSet.RestConfig, mgrSet.CommandExecutor),
		doguCommandExecutor:   mgrSet.CommandExecutor,
	}
}

// Upgrade executes all necessary steps to update a dogu to a new version.
func (ue *upgradeExecutor) Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error {
	ue.normalEventf(toDoguResource, "Registering upgraded version %s in local dogu registry...", toDogu.Version)
	err := registerUpgradedDoguVersion(ue.doguRegistrator, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Registering optional service accounts...")
	err = registerNewServiceAccount(ctx, ue.serviceAccountCreator, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Pulling new image %s:%s...", toDogu.Image, toDogu.Version)
	imageConfigFile, err := pullUpgradeImage(ctx, ue.imageRegistry, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Updating dogu resources in the cluster...")
	err = ue.updateDoguResources(ctx, ue.resourceUpserter, toDoguResource, toDogu, fromDogu, imageConfigFile)
	if err != nil {
		return err
	}

	err = ue.applyPostUpgradeScript(ctx, toDoguResource, fromDogu, toDogu)
	if err != nil {
		return fmt.Errorf("post-upgrade failed: %w", err)
	}

	ue.normalEventf(toDoguResource, "Reverting to original startup probe values...")
	err = revertStartupProbeAfterUpdate(ctx, toDoguResource, toDogu, ue.client)
	if err != nil {
		return err
	}

	return nil
}

func increaseStartupProbeTimeoutForUpdate(containerName string, deployment *appsv1.Deployment) {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName && deployment.Spec.Template.Spec.Containers[i].StartupProbe != nil {
			deployment.Spec.Template.Spec.Containers[i].StartupProbe.FailureThreshold = upgradeStartupProbeFailureThresholdRetries
			break
		}
	}
}

func revertStartupProbeAfterUpdate(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, client client.Client) error {
	originalStartupProbe := resource.CreateStartupProbe(toDogu)

	// We often used to have resource conflicts here, because the API server wasn't fast enough.
	// This mechanism retries the operation if there is a conflict.
	err := retry.OnConflict(func() error {
		deployment, err := toDoguResource.GetDeployment(ctx, client)
		if err != nil {
			return err
		}

		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == toDoguResource.Name && container.StartupProbe != nil {
				deployment.Spec.Template.Spec.Containers[i].StartupProbe = originalStartupProbe
				break
			}
		}

		return client.Update(ctx, deployment)
	})
	if err != nil {
		return err
	}

	return nil
}

func registerUpgradedDoguVersion(cesreg cloudogu.DoguRegistrator, toDogu *core.Dogu) error {
	err := cesreg.RegisterDoguVersion(toDogu)
	if err != nil {
		return fmt.Errorf("failed to register upgrade: %w", err)
	}

	return nil
}

func registerNewServiceAccount(ctx context.Context, saCreator cloudogu.ServiceAccountCreator, toDogu *core.Dogu) error {
	err := saCreator.CreateAll(ctx, toDogu)
	if err != nil {
		if err != nil {
			return fmt.Errorf("failed to register service accounts: %w", err)
		}
	}
	return nil
}

func pullUpgradeImage(ctx context.Context, imgRegistry cloudogu.ImageRegistry, toDogu *core.Dogu) (*imagev1.ConfigFile, error) {
	configFile, err := imgRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to pull upgrade image: %w", err)
	}

	return configFile, nil
}

func (ue *upgradeExecutor) applyCustomK8sScripts(ctx context.Context, toDoguResource *k8sv1.Dogu, execPod cloudogu.ExecPod) error {
	var customK8sResources map[string]string
	customK8sResources, err := extractCustomK8sResources(ctx, ue.k8sFileExtractor, execPod)
	if err != nil {
		return err
	}

	if len(customK8sResources) > 0 {
		ue.normalEventf(toDoguResource, "Applying/Updating custom dogu resources to the cluster: [%s]", util.GetMapKeysAsString(customK8sResources))
	}

	return applyCustomK8sResources(ctx, ue.collectApplier, toDoguResource, customK8sResources)
}

func extractCustomK8sResources(ctx context.Context, extractor cloudogu.FileExtractor, execPod cloudogu.ExecPod) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, execPod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier cloudogu.CollectApplier, toDoguResource *k8sv1.Dogu, customK8sResources map[string]string) error {
	err := collectApplier.CollectApply(ctx, customK8sResources, toDoguResource)
	if err != nil {
		return fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu, execPod cloudogu.ExecPod) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPreUpgrade) {
		return nil
	}

	preUpgradeScriptCmd := toDogu.GetExposedCommand(core.ExposedCommandPreUpgrade)

	ue.normalEventf(toDoguResource, "Copying optional pre-upgrade scripts...")

	fromDoguPod, err := getPodForDogu(ctx, ue.client, fromDogu)
	if err != nil {
		return fmt.Errorf("failed to find pod for dogu %s:%s : %w", fromDogu.GetSimpleName(), fromDogu.Version, err)
	}

	err = ue.copyPreUpgradeScriptFromPodToPod(ctx, execPod, fromDoguPod, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Applying optional pre-upgrade scripts...")
	err = ue.applyPreUpgradeScriptToOlderDogu(ctx, fromDogu, fromDoguPod, toDoguResource, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	return nil
}

func (ue *upgradeExecutor) copyPreUpgradeScriptFromPodToPod(ctx context.Context, srcPod cloudogu.ExecPod, destPod *corev1.Pod, preUpgradeScriptCmd *core.ExposedCommand) error {
	tarCommand := exec.NewShellCommand("/bin/tar", "cf", "-", preUpgradeScriptCmd.Command)
	archive, err := srcPod.Exec(ctx, tarCommand)
	if err != nil {
		return fmt.Errorf("failed to get pre-upgrade script from execpod with command '%s', stdout: '%s':  %w", tarCommand.String(), archive, err)
	}

	createPathCommand := exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)
	out, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, destPod, createPathCommand, cloudogu.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to create pre-upgrade target dir with command '%s', stdout: '%s': %w", createPathCommand.String(), out, err)
	}

	untarCommand := exec.NewShellCommandWithStdin(archive, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)
	out, err = ue.doguCommandExecutor.ExecCommandForPod(ctx, destPod, untarCommand, cloudogu.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to extract pre-upgrade script to dogu pod with command '%s', stdout: '%s': %w", untarCommand.String(), out, err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScriptToOlderDogu(
	ctx context.Context,
	fromDogu *core.Dogu,
	fromDoguPod *corev1.Pod,
	toDoguResource *k8sv1.Dogu,
	preUpgradeCmd *core.ExposedCommand,
) error {
	logger := log.FromContext(ctx)
	logger.Info("applying pre-upgrade script to old dogu")

	preUpgradeScriptPath := filepath.Join(preUpgradeScriptDir, filepath.Base(preUpgradeCmd.Command))

	preUpgradeShellCmd := exec.NewShellCommand(preUpgradeScriptPath, fromDogu.Version, toDoguResource.Spec.Version)

	logger.Info("Executing pre-upgrade command " + preUpgradeShellCmd.String())
	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, fromDoguPod, preUpgradeShellCmd, cloudogu.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func getPodForDogu(ctx context.Context, cli client.Client, dogu *core.Dogu) (*corev1.Pod, error) {
	fromDoguLabels := map[string]string{
		k8sv1.DoguLabelName:    dogu.GetSimpleName(),
		k8sv1.DoguLabelVersion: dogu.Version,
	}

	return k8sv1.GetPodForLabels(ctx, cli, fromDoguLabels)
}

func (ue *upgradeExecutor) applyPostUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPostUpgrade) {
		return nil
	}

	postUpgradeCmd := toDogu.GetExposedCommand(core.ExposedCommandPostUpgrade)

	ue.normalEventf(toDoguResource, "Applying optional post-upgrade scripts...")
	return ue.executePostUpgradeScript(ctx, toDoguResource, fromDogu, postUpgradeCmd)
}

func (ue *upgradeExecutor) executePostUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *core.Dogu, postUpgradeCmd *core.ExposedCommand) error {
	postUpgradeShellCmd := exec.NewShellCommand(postUpgradeCmd.Command, fromDogu.Version, toDoguResource.Spec.Version)

	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, toDoguResource, postUpgradeShellCmd, cloudogu.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) updateDoguResources(ctx context.Context, upserter cloudogu.ResourceUpserter, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, fromDogu *core.Dogu, image *imagev1.ConfigFile) error {
	_, err := upserter.UpsertDoguService(ctx, toDoguResource, image)
	if err != nil {
		return err
	}

	_, err = upserter.UpsertDoguExposedService(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Extracting optional custom K8s resources...")
	execPod, err := ue.execPodFactory.NewExecPod(toDoguResource, toDogu)
	if err != nil {
		return err
	}
	err = execPod.Create(ctx)
	if err != nil {
		return err
	}
	defer deleteExecPod(ctx, execPod, ue.eventRecorder, toDoguResource)

	err = ue.applyPreUpgradeScript(ctx, toDoguResource, fromDogu, toDogu, execPod)
	if err != nil {
		return fmt.Errorf("pre-upgrade failed: %w", err)
	}

	err = ue.applyCustomK8sScripts(ctx, toDoguResource, execPod)
	if err != nil {
		return err
	}

	_, err = upserter.UpsertDoguDeployment(
		ctx,
		toDoguResource,
		toDogu,
		func(deployment *appsv1.Deployment) {
			increaseStartupProbeTimeoutForUpdate(toDoguResource.Name, deployment)
		},
	)
	if err != nil {
		return err
	}

	// Set the health status to 'unavailable' early, to prevent setting the new installed version while the health
	// status is still 'available' (which would lead to a false healthy upgrade being displayed).
	err = ue.setHealthStatusUnavailable(ctx, toDoguResource, err)
	if err != nil {
		return err
	}

	_, err = upserter.UpsertDoguPVCs(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	return nil
}

func (ue *upgradeExecutor) setHealthStatusUnavailable(ctx context.Context, toDoguResource *k8sv1.Dogu, err error) error {
	err = toDoguResource.UpdateStatusWithRetry(ctx, ue.client, func(d *k8sv1.Dogu) { d.Status.Health = k8sv1.UnavailableHealthStatus })
	if err != nil {
		message := fmt.Sprintf("failed to update dogu %q with health status %q", toDoguResource.Spec.Name, toDoguResource.Status.Health)
		ue.eventRecorder.Event(toDoguResource, corev1.EventTypeWarning, EventReason, message)
		return fmt.Errorf("%s: %w", message, err)
	}
	ue.normalEventf(toDoguResource, "Successfully updated health status to %q", toDoguResource.Status.Health)
	return nil
}

func (ue *upgradeExecutor) normalEventf(doguResource *k8sv1.Dogu, msg string, args ...interface{}) {
	ue.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, msg, args...)
}

func deleteExecPod(ctx context.Context, execPod cloudogu.ExecPod, recorder record.EventRecorder, doguResource *k8sv1.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
