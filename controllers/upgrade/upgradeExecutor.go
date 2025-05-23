package upgrade

import (
	"context"
	"fmt"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/retry-lib/retry"
)

const (
	EventReason                     = "Upgrading"
	ErrorOnFailedUpgradeEventReason = "ErrUpgrade"
)

// upgradeStartupProbeFailureThresholdRetries contains the number of times how often a startup probe may fail. This
// value will be multiplied with 10 seconds for each timeout so that f. i. 1080 timeouts lead to a threshold of 3 hours.
const upgradeStartupProbeFailureThresholdRetries = int32(1080)

const preUpgradeScriptDir = "/tmp/pre-upgrade"

const maxRetries = 20

type upgradeExecutor struct {
	client                client.Client
	ecosystemClient       doguClient.EcoSystemV2Interface
	eventRecorder         record.EventRecorder
	imageRegistry         imageRegistry
	collectApplier        resource.CollectApplier
	k8sFileExtractor      exec.FileExtractor
	serviceAccountCreator serviceAccountCreator
	doguRegistrator       doguRegistrator
	resourceUpserter      resource.ResourceUpserter
	execPodFactory        exec.ExecPodFactory
	doguCommandExecutor   exec.CommandExecutor
}

// NewUpgradeExecutor creates a new upgrade executor.
func NewUpgradeExecutor(
	client client.Client,
	mgrSet *util.ManagerSet,
	eventRecorder record.EventRecorder,
	ecosystemClient doguClient.EcoSystemV2Interface,
) *upgradeExecutor {
	return &upgradeExecutor{
		client:                client,
		ecosystemClient:       ecosystemClient,
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
func (ue *upgradeExecutor) Upgrade(ctx context.Context, toDoguResource *doguv2.Dogu, fromDogu, toDogu *core.Dogu) error {
	ue.normalEventf(toDoguResource, "Registering upgraded version %s in local dogu registry...", toDogu.Version)
	err := registerUpgradedDoguVersion(ctx, ue.doguRegistrator, toDogu)
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

func revertStartupProbeAfterUpdate(ctx context.Context, toDoguResource *doguv2.Dogu, toDogu *core.Dogu, client client.Client) error {
	originalStartupProbe := resource.CreateStartupProbe(toDogu)
	shouldWaitForNewPod := false

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
				shouldWaitForNewPod = true
				return client.Update(ctx, deployment)
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !shouldWaitForNewPod {
		return nil
	}

	// The update will trigger a pod restart. We wait here for the pod to ensure that the dogu operator does not install a next dogu
	// with potential service account create command for this upgraded dogu. The restart could lead to errors in the service account creation.
	return waitForPodWithRevertedStartupProbe(ctx, client, toDoguResource, originalStartupProbe)
}

func waitForPodWithRevertedStartupProbe(ctx context.Context, client client.Client, toDoguResource *doguv2.Dogu, probe *corev1.Probe) error {
	return retry.OnError(maxRetries, retry.AlwaysRetryFunc, func() error {
		log.FromContext(ctx).Info(fmt.Sprintf("Wait for %s pod with reverted startup probe", toDoguResource.Name))
		pod, getPodErr := toDoguResource.GetPod(ctx, client)
		if getPodErr != nil {
			return getPodErr
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == toDoguResource.Name && container.StartupProbe != nil {
				// We only edit the FailureThreshold. So check only this attribute.
				if container.StartupProbe.FailureThreshold == probe.FailureThreshold {
					return nil
				}
			}
		}

		return fmt.Errorf("retry: pod %s of dogu %s does not have original startup probe", pod.Name, toDoguResource.Name)
	})
}

func registerUpgradedDoguVersion(ctx context.Context, cesreg doguRegistrator, toDogu *core.Dogu) error {
	err := cesreg.RegisterDoguVersion(ctx, toDogu)
	if err != nil {
		return fmt.Errorf("failed to register upgrade: %w", err)
	}

	return nil
}

func registerNewServiceAccount(ctx context.Context, saCreator serviceAccountCreator, toDogu *core.Dogu) error {
	err := saCreator.CreateAll(ctx, toDogu)
	if err != nil {
		return fmt.Errorf("failed to register service accounts: %w", err)
	}
	return nil
}

func pullUpgradeImage(ctx context.Context, imgRegistry imageRegistry, toDogu *core.Dogu) (*imagev1.ConfigFile, error) {
	configFile, err := imgRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to pull upgrade image: %w", err)
	}

	return configFile, nil
}

func (ue *upgradeExecutor) applyCustomK8sScripts(ctx context.Context, toDoguResource *doguv2.Dogu, execPod exec.ExecPod) error {
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

func extractCustomK8sResources(ctx context.Context, extractor exec.FileExtractor, execPod exec.ExecPod) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, execPod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier resource.CollectApplier, toDoguResource *doguv2.Dogu, customK8sResources map[string]string) error {
	err := collectApplier.CollectApply(ctx, customK8sResources, toDoguResource)
	if err != nil {
		return fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScript(ctx context.Context, toDoguResource *doguv2.Dogu, fromDogu, toDogu *core.Dogu, execPod exec.ExecPod) error {
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

func (ue *upgradeExecutor) copyPreUpgradeScriptFromPodToPod(ctx context.Context, srcPod exec.ExecPod, destPod *corev1.Pod, preUpgradeScriptCmd *core.ExposedCommand) error {
	tarCommand := exec.NewShellCommand("/bin/tar", "cf", "-", preUpgradeScriptCmd.Command)
	archive, err := srcPod.Exec(ctx, tarCommand)
	if err != nil {
		return fmt.Errorf("failed to get pre-upgrade script from execpod with command '%s', stdout: '%s':  %w", tarCommand.String(), archive, err)
	}

	createPathCommand := exec.NewShellCommand("/bin/mkdir", "-p", preUpgradeScriptDir)
	out, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, destPod, createPathCommand, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to create pre-upgrade target dir with command '%s', stdout: '%s': %w", createPathCommand.String(), out, err)
	}

	untarCommand := exec.NewShellCommandWithStdin(archive, "/bin/tar", "xf", "-", "-C", preUpgradeScriptDir)
	out, err = ue.doguCommandExecutor.ExecCommandForPod(ctx, destPod, untarCommand, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to extract pre-upgrade script to dogu pod with command '%s', stdout: '%s': %w", untarCommand.String(), out, err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScriptToOlderDogu(
	ctx context.Context,
	fromDogu *core.Dogu,
	fromDoguPod *corev1.Pod,
	toDoguResource *doguv2.Dogu,
	preUpgradeCmd *core.ExposedCommand,
) error {
	logger := log.FromContext(ctx)
	logger.Info("applying pre-upgrade script to old dogu")

	preUpgradeScriptPath := filepath.Join(preUpgradeScriptDir, filepath.Base(preUpgradeCmd.Command))

	preUpgradeShellCmd := exec.NewShellCommand(preUpgradeScriptPath, fromDogu.Version, toDoguResource.Spec.Version)

	logger.Info("Executing pre-upgrade command " + preUpgradeShellCmd.String())
	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, fromDoguPod, preUpgradeShellCmd, exec.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func getPodForDogu(ctx context.Context, cli client.Client, dogu *core.Dogu) (*corev1.Pod, error) {
	fromDoguLabels := map[string]string{
		doguv2.DoguLabelName:    dogu.GetSimpleName(),
		doguv2.DoguLabelVersion: dogu.Version,
	}

	return doguv2.GetPodForLabels(ctx, cli, fromDoguLabels)
}

func (ue *upgradeExecutor) applyPostUpgradeScript(ctx context.Context, toDoguResource *doguv2.Dogu, fromDogu, toDogu *core.Dogu) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPostUpgrade) {
		return nil
	}

	postUpgradeCmd := toDogu.GetExposedCommand(core.ExposedCommandPostUpgrade)

	ue.normalEventf(toDoguResource, "Applying optional post-upgrade scripts...")
	return ue.executePostUpgradeScript(ctx, toDoguResource, fromDogu, postUpgradeCmd)
}

func (ue *upgradeExecutor) executePostUpgradeScript(ctx context.Context, toDoguResource *doguv2.Dogu, fromDogu *core.Dogu, postUpgradeCmd *core.ExposedCommand) error {
	postUpgradeShellCmd := exec.NewShellCommand(postUpgradeCmd.Command, fromDogu.Version, toDoguResource.Spec.Version)

	toDoguPod := &corev1.Pod{}
	// Wait until new pod is spawned
	err := retry.OnError(maxRetries, retry.TestableRetryFunc, func() error {
		var getPodErr error
		toDoguPod, getPodErr = toDoguResource.GetPod(ctx, ue.client)
		if getPodErr != nil {
			return &retry.TestableRetrierError{Err: fmt.Errorf("failed to get new %s pod for post upgrade: %w", toDoguResource.Name, getPodErr)}
		}

		return nil
	})
	if err != nil {
		return err
	}

	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, toDoguPod, postUpgradeShellCmd, exec.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) updateDoguResources(ctx context.Context, upserter resource.ResourceUpserter, toDoguResource *doguv2.Dogu, toDogu *core.Dogu, fromDogu *core.Dogu, image *imagev1.ConfigFile) error {
	_, err := upserter.UpsertDoguService(ctx, toDoguResource, toDogu, image)
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
	err = ue.setHealthStatusUnavailable(ctx, toDoguResource)
	if err != nil {
		return err
	}

	_, err = upserter.UpsertDoguPVCs(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	err = upserter.UpsertDoguNetworkPolicies(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	return nil
}

func (ue *upgradeExecutor) setHealthStatusUnavailable(ctx context.Context, toDoguResource *doguv2.Dogu) error {
	toDoguResource, err := ue.ecosystemClient.Dogus(toDoguResource.Namespace).UpdateStatusWithRetry(ctx, toDoguResource,
		func(status doguv2.DoguStatus) doguv2.DoguStatus {
			status.Health = doguv2.UnavailableHealthStatus
			return status
		}, metav1.UpdateOptions{})
	if err != nil {
		message := fmt.Sprintf("failed to update dogu %q with health status %q", toDoguResource.Spec.Name, doguv2.UnavailableHealthStatus)
		ue.eventRecorder.Event(toDoguResource, corev1.EventTypeWarning, EventReason, message)
		return fmt.Errorf("%s: %w", message, err)
	}
	ue.normalEventf(toDoguResource, "Successfully updated health status to %q", toDoguResource.Status.Health)
	return nil
}

func (ue *upgradeExecutor) normalEventf(doguResource *doguv2.Dogu, msg string, args ...interface{}) {
	ue.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, msg, args...)
}

func deleteExecPod(ctx context.Context, execPod exec.ExecPod, recorder record.EventRecorder, doguResource *doguv2.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
