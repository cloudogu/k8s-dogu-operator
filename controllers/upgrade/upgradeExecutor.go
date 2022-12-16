package upgrade

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal"
)

const (
	EventReason                     = "Upgrading"
	ErrorOnFailedUpgradeEventReason = "ErrUpgrade"
)

// upgradeStartupProbeFailureThresholdRetries contains the number of times how often a startup probe may fail. This
// value will be multiplied with 10 seconds for each timeout so that f. i. 1080 timeouts lead to a threshold of 3 hours.
const upgradeStartupProbeFailureThresholdRetries = int32(1080)

type upgradeExecutor struct {
	client                client.Client
	eventRecorder         record.EventRecorder
	imageRegistry         internal.ImageRegistry
	collectApplier        internal.CollectApplier
	k8sFileExtractor      internal.FileExtractor
	serviceAccountCreator internal.ServiceAccountCreator
	doguRegistrator       internal.DoguRegistrator
	resourceUpserter      internal.ResourceUpserter
	execPodFactory        internal.ExecPodFactory
	doguCommandExecutor   internal.CommandExecutor
}

// NewUpgradeExecutor creates a new upgrade executor.
func NewUpgradeExecutor(
	client client.Client,
	config *rest.Config,
	commandExecutor internal.CommandExecutor,
	eventRecorder record.EventRecorder,
	imageRegistry internal.ImageRegistry,
	collectApplier internal.CollectApplier,
	k8sFileExtractor internal.FileExtractor,
	serviceAccountCreator internal.ServiceAccountCreator,
	registry registry.Registry,
) *upgradeExecutor {
	doguReg := cesregistry.NewCESDoguRegistrator(client, registry, nil)
	limitPatcher := limit.NewDoguDeploymentLimitPatcher(registry)
	upserter := resource.NewUpserter(client, limitPatcher)

	return &upgradeExecutor{
		client:                client,
		eventRecorder:         eventRecorder,
		imageRegistry:         imageRegistry,
		collectApplier:        collectApplier,
		k8sFileExtractor:      k8sFileExtractor,
		serviceAccountCreator: serviceAccountCreator,
		doguRegistrator:       doguReg,
		resourceUpserter:      upserter,
		execPodFactory:        exec.NewExecPodFactory(client, config, commandExecutor),
		doguCommandExecutor:   commandExecutor,
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

	deployment := &appsv1.Deployment{}
	err := client.Get(ctx, toDoguResource.GetObjectKey(), deployment)
	if err != nil {
		return err
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == toDoguResource.Name && container.StartupProbe != nil {
			deployment.Spec.Template.Spec.Containers[i].StartupProbe = originalStartupProbe
			break
		}
	}

	err = client.Update(ctx, deployment)
	if err != nil {
		return err
	}

	return nil
}

func registerUpgradedDoguVersion(cesreg internal.DoguRegistrator, toDogu *core.Dogu) error {
	err := cesreg.RegisterDoguVersion(toDogu)
	if err != nil {
		return fmt.Errorf("failed to register upgrade: %w", err)
	}

	return nil
}

func registerNewServiceAccount(ctx context.Context, saCreator internal.ServiceAccountCreator, toDogu *core.Dogu) error {
	err := saCreator.CreateAll(ctx, toDogu)
	if err != nil {
		if err != nil {
			return fmt.Errorf("failed to register service accounts: %w", err)
		}
	}
	return nil
}

func pullUpgradeImage(ctx context.Context, imgRegistry internal.ImageRegistry, toDogu *core.Dogu) (*imagev1.ConfigFile, error) {
	configFile, err := imgRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to pull upgrade image: %w", err)
	}

	return configFile, nil
}

func (ue *upgradeExecutor) applyCustomK8sScripts(ctx context.Context, toDoguResource *k8sv1.Dogu, execPod internal.ExecPod) error {
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

func extractCustomK8sResources(ctx context.Context, extractor internal.FileExtractor, execPod internal.ExecPod) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, execPod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier internal.CollectApplier, toDoguResource *k8sv1.Dogu, customK8sResources map[string]string) error {
	err := collectApplier.CollectApply(ctx, customK8sResources, toDoguResource)
	if err != nil {
		return fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu, execPod internal.ExecPod) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPreUpgrade) {
		return nil
	}

	preUpgradeScriptCmd := toDogu.GetExposedCommand(core.ExposedCommandPreUpgrade)

	ue.normalEventf(toDoguResource, "Copying optional pre-upgrade scripts...")
	err := copyPreUpgradeScriptFromPodToPod(ctx, execPod, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Applying optional pre-upgrade scripts...")
	err = ue.applyPreUpgradeScriptToOlderDogu(ctx, fromDogu, toDoguResource, preUpgradeScriptCmd)
	if err != nil {
		return err
	}

	return nil
}

func copyPreUpgradeScriptFromPodToPod(ctx context.Context, execPod internal.ExecPod, preUpgradeScriptCmd *core.ExposedCommand) error {
	// the exec pod is based on the image-to-be-upgraded. Thus, upgrade scripts should have the right executable
	// permissions which should be retained during the file copy.
	const copyCmd = "/bin/cp"
	// since we copy to a directory we can here ignore any included directories in the command
	// example: copy from /resource/myscript.sh to /dogu-reserved/myscript.sh
	copyPreUpgradeScriptCmd := exec.NewShellCommand(copyCmd, preUpgradeScriptCmd.Command, resource.DoguReservedPath)

	out, err := execPod.Exec(ctx, copyPreUpgradeScriptCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s' in execpod, stdout: '%s':  %w", copyPreUpgradeScriptCmd.String(), out, err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScriptToOlderDogu(ctx context.Context, fromDogu *core.Dogu, toDoguResource *k8sv1.Dogu, preUpgradeCmd *core.ExposedCommand) error {
	logger := log.FromContext(ctx)
	logger.Info("applying pre-upgrade script to old dogu")

	fromDoguPod, err := getPodNameForFromDogu(ctx, ue.client, fromDogu)
	if err != nil {
		return fmt.Errorf("failed to find fromDoguPod for dogu %s:%s : %w", fromDogu.GetSimpleName(), fromDogu.Version, err)
	}

	return ue.executePreUpgradeScript(ctx, fromDoguPod, preUpgradeCmd, fromDogu.Version, toDoguResource.Spec.Version)
}

func (ue *upgradeExecutor) executePreUpgradeScript(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand, fromVersion, toVersion string) error {
	logger := log.FromContext(ctx)
	scriptPath := fmt.Sprintf("%s%s", resource.DoguReservedPath, cmd.Command)
	// finally execute the copied pre-upgrade script, due to the dogu-reserved location relative file paths do not work
	preUpgradeCmd := exec.NewShellCommand(scriptPath, fromVersion, toVersion)

	logger.Info("Executing pre-upgrade command " + preUpgradeCmd.String())
	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, fromPod, preUpgradeCmd, internal.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeCmd, outBuf, err)
	}

	return nil
}

func getPodNameForFromDogu(ctx context.Context, cli client.Client, fromDogu *core.Dogu) (*corev1.Pod, error) {
	fromDoguLabels := map[string]string{
		k8sv1.DoguLabelName:    fromDogu.GetSimpleName(),
		k8sv1.DoguLabelVersion: fromDogu.Version,
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

	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, toDoguResource, postUpgradeShellCmd, internal.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) updateDoguResources(ctx context.Context, upserter internal.ResourceUpserter, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, fromDogu *core.Dogu, image *imagev1.ConfigFile) error {
	_, err := upserter.UpsertDoguService(ctx, toDoguResource, image)
	if err != nil {
		return err
	}

	_, err = upserter.UpsertDoguExposedServices(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Extracting optional custom K8s resources...")
	execPod, err := ue.execPodFactory.NewExecPod(internal.VolumeModeUpgrade, toDoguResource, toDogu)
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

	_, err = upserter.UpsertDoguPVCs(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	return nil
}

func (ue *upgradeExecutor) normalEventf(doguResource *k8sv1.Dogu, msg string, args ...interface{}) {
	ue.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, msg, args...)
}

func deleteExecPod(ctx context.Context, execPod internal.ExecPod, recorder record.EventRecorder, doguResource *k8sv1.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
