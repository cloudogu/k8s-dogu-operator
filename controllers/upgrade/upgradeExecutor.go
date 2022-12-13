package upgrade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/client-go/rest"
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

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventReason                     = "Upgrading"
	ErrorOnFailedUpgradeEventReason = "ErrUpgrade"
)

const (
	preUpgradeExecStrategyNative = iota
	preUpgradeExecStrategyShellPipe
)

type preUpgradeExecStrategy int

// copyErrorDetectorCantCreateFile recognizes at least two different errors:
// unwritable target directory and unoverwritable script file
var copyErrorDetectorCantCreateFile, _ = regexp.Compile(".*cp: can't create '.+':.*")

func isPreUpgradeExecErrRetryable(err error) bool {
	return copyErrorDetectorCantCreateFile.MatchString(err.Error())
}

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
		return fmt.Errorf("post-upgrade failed :%w", err)
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
		if container.Name == containerName {
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
		if container.Name == toDoguResource.Name {
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

	pod, err := getPodNameForFromDogu(ctx, ue.client, fromDogu)
	if err != nil {
		return fmt.Errorf("failed to find pod for dogu %s:%s : %w", fromDogu.GetSimpleName(), fromDogu.Version, err)
	}

	// possibly create the necessary directory structure for the following copy action
	err = ue.createMissingUpgradeDirs(ctx, pod, preUpgradeCmd)
	if err != nil {
		return err
	}

	usePreUpgradeExecStrategy := preUpgradeExecStrategy(preUpgradeExecStrategyNative)
	err = ue.copyPreUpgradeScriptToDoguReserved(ctx, pod, preUpgradeCmd)
	if err != nil {
		if !isPreUpgradeExecErrRetryable(err) {
			return fmt.Errorf("error while executing pre-upgrade script: %w", err)
		}
		ue.normalEventf(toDoguResource, "Running pre-upgrade script with shell pipe strategy instead...")
		// retry with a different strategy because we are certain that the upgrade script could not be copied because
		// of file system permissions.
		logger.Info(fmt.Sprintf("Native pre-upgrade script execution failed with '%s': Retry with strategy %d", err.Error(), usePreUpgradeExecStrategy))
		usePreUpgradeExecStrategy = preUpgradeExecStrategy(preUpgradeExecStrategyShellPipe)
	}

	return ue.executePreUpgradeScript(ctx, pod, preUpgradeCmd, fromDogu.Version, toDoguResource.Spec.Version, usePreUpgradeExecStrategy)
}

func (ue *upgradeExecutor) copyPreUpgradeScriptToDoguReserved(ctx context.Context, pod *corev1.Pod, preUpgradeCmd *core.ExposedCommand) error {
	preUpgradeBackCopyCmd := getPreUpgradeBackCopyCmd(preUpgradeCmd)

	// copy the file back to the directory where it resided in the upgrade image
	// example: /dogu-reserved/myscript.sh to /resource/myscript.sh
	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, pod, preUpgradeBackCopyCmd, internal.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeBackCopyCmd, outBuf, err)
	}

	return nil
}

func getPreUpgradeBackCopyCmd(preUpgradeCmd *core.ExposedCommand) internal.ShellCommand {
	_, fileName := filepath.Split(preUpgradeCmd.Command)
	filePathInDoguReserved := filepath.Join(resource.DoguReservedPath, fileName)

	return exec.NewShellCommand("/bin/cp", filePathInDoguReserved, preUpgradeCmd.Command)
}

func (ue *upgradeExecutor) createMissingUpgradeDirs(ctx context.Context, pod *corev1.Pod, cmd *core.ExposedCommand) error {
	baseDir, _ := filepath.Split(cmd.Command)

	mkdirCmd := exec.NewShellCommand("/bin/mkdir", "-p", baseDir)

	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, pod, mkdirCmd, internal.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", mkdirCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) executePreUpgradeScript(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand, fromVersion, toVersion string, strategy preUpgradeExecStrategy) error {

	switch strategy {
	case preUpgradeExecStrategyNative:
		return ue.execPreUpgradeExecNatively(ctx, fromPod, cmd, fromVersion, toVersion)
	case preUpgradeExecStrategyShellPipe:
		return ue.execPreUpgradeExecShellPipe(ctx, fromPod, cmd, fromVersion, toVersion)
	default:
		return fmt.Errorf("unsupported pre-upgrade execution value %d found", strategy)
	}
}

func (ue *upgradeExecutor) execPreUpgradeExecNatively(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand, fromVersion, toVersion string) error {
	logger := log.FromContext(ctx)
	// the previous steps took great lengths to copy the pre-upgrade script to the original place
	// so we can finally execute it as described by the dogu.json.
	preUpgradeCmd := exec.NewShellCommand(cmd.Command, fromVersion, toVersion)

	logger.Info("Executing pre-upgrade command " + preUpgradeCmd.String())
	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, fromPod, preUpgradeCmd, internal.PodReady)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeCmd, outBuf, err)
	}

	return nil
}

// execPreUpgradeExecShellPipe executes the pre-upgrade script in a different way than just calling the previously
// copied script file (which seemingly did not work since we ended up here). This method basically copies the whole
// script content into a pipe and tries to exec it against a suitable shell interpreter. This approach does not work for
// pre-upgrade scripts with a sticky bit because the content will be executed as the user that is currently running in
// the dogu container, and said current user is most likely non-privileged.
func (ue *upgradeExecutor) execPreUpgradeExecShellPipe(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand, fromVersion, toVersion string) error {
	logger := log.FromContext(ctx)
	logger.Info("Re-executing pre-upgrade command")
	errMsg := fmt.Sprintf("failed to re-execute pre-upgrade script '%s': output: ", cmd.Command)

	interpreterOut, err := ue.detectPreUpgradeScriptInterpreter(ctx, fromPod, cmd)
	if err != nil {
		return fmt.Errorf(errMsg+"'%s': %w", interpreterOut, err)
	}

	outBuf, err := ue.execPreUpgradeShellPipeWithInterpreter(ctx, fromPod, cmd, fromVersion, toVersion, interpreterOut)
	if err != nil {
		return fmt.Errorf(errMsg+"'%s': %w", outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) detectPreUpgradeScriptInterpreter(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand) (string, error) {
	scriptPath := fmt.Sprintf("%s%s", resource.DoguReservedPath, cmd.Command)

	// every container should provide any form of shell which would alias to sh
	preUpgradeCmd := exec.NewShellCommand("/bin/grep", "'#!'", scriptPath)

	outBuf, err := ue.doguCommandExecutor.ExecCommandForPod(ctx, fromPod, preUpgradeCmd, internal.PodReady)
	if err != nil {
		return outBuf.String(), err
	}

	interpreter, err := getScriptInterpreterFromOutput(outBuf.String())
	if err != nil {
		return "", fmt.Errorf("failed to detect script interpreter in %s: %w", scriptPath, err)
	}

	return interpreter, nil
}

func getScriptInterpreterFromOutput(interpreterOut string) (string, error) {
	split := strings.Split(interpreterOut, "#!")
	if len(split) < 2 {
		return "", errors.New("shebang line not found")
	}

	interpreter := strings.TrimSpace(split[1])
	if !strings.HasPrefix(interpreter, "/") {
		return "", fmt.Errorf("shebang does not look like path to an executable: %s", interpreter)
	}

	return interpreter, nil
}

func (ue *upgradeExecutor) execPreUpgradeShellPipeWithInterpreter(ctx context.Context, fromPod *corev1.Pod, cmd *core.ExposedCommand, fromVersion, toVersion, interpreter string) (*bytes.Buffer, error) {
	// "cd dirname" ensures to enter the correct directory to avoid problems with script relative filenames.
	// the second part pipes the script from the dogu-reservation into bash along with the necessary dogu versions
	shellPipeCmd := fmt.Sprintf(`cd $(dirname %s) && (cat %s%s | %s -s "%s" "%s")`,
		cmd.Command, resource.DoguReservedPath, cmd.Command, interpreter, fromVersion, toVersion)
	preUpgradeCmd := exec.NewShellCommand("/bin/sh", "-c", shellPipeCmd)

	return ue.doguCommandExecutor.ExecCommandForPod(ctx, fromPod, preUpgradeCmd, internal.PodReady)
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
		return fmt.Errorf("pre-upgrade failed :%w", err)
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
