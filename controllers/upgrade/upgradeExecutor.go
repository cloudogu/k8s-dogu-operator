package upgrade

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/rest"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

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

const upgradeFailureThreshold = int32(60)

type upgradeExecutor struct {
	client                client.Client
	eventRecorder         record.EventRecorder
	imageRegistry         imageRegistry
	collectApplier        collectApplier
	k8sFileExtractor      fileExtractor
	serviceAccountCreator serviceAccountCreator
	doguRegistrator       doguRegistrator
	resourceUpserter      resourceUpserter
	execPodFactory        execPodFactory
	doguCommandExecutor   commandExecutor
}

// NewUpgradeExecutor creates a new upgrade executor.
func NewUpgradeExecutor(
	client client.Client,
	config *rest.Config,
	commandExecutor commandExecutor,
	eventRecorder record.EventRecorder,
	imageRegistry imageRegistry,
	collectApplier collectApplier,
	k8sFileExtractor fileExtractor,
	serviceAccountCreator serviceAccountCreator,
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
		execPodFactory:        util.NewExecPodFactory(client, config, commandExecutor),
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
	err = registerNewServiceAccount(ctx, ue.serviceAccountCreator, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Pulling new image %s:%s...", toDogu.Image, toDogu.Version)
	imageConfigFile, err := pullUpgradeImage(ctx, ue.imageRegistry, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Extracting optional custom K8s resources...")
	execPod, err := ue.execPodFactory.NewExecPod(util.ExecPodVolumeModeUpgrade, toDoguResource, toDogu)
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
		return err
	}

	customDeployment, err := ue.applyCustomK8sScripts(ctx, toDoguResource, execPod)
	if err != nil {
		return err
	}

	// change startup probe timeout to 10 min
	customDeployment = increaseStartupProbeTimeoutForUpdate(toDoguResource.Name, customDeployment)

	ue.normalEventf(toDoguResource, "Updating dogu resources in the cluster...")
	err = updateDoguResources(ctx, ue.resourceUpserter, toDoguResource, toDogu, imageConfigFile, customDeployment)
	if err != nil {
		return err
	}

	err = ue.applyPostUpgradeScript(ctx, toDoguResource, fromDogu, toDogu)
	if err != nil {
		return err
	}

	err = revertStartupProbeAfterUpdate(ctx, toDoguResource, toDogu, ue.client)
	if err != nil {
		return err
	}

	return nil
}

func increaseStartupProbeTimeoutForUpdate(containerName string, customDeployment *appsv1.Deployment) *appsv1.Deployment {
	container := corev1.Container{
		Name: containerName,
		StartupProbe: &corev1.Probe{
			FailureThreshold: upgradeFailureThreshold,
		},
	}
	if customDeployment == nil {
		customDeployment = &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{container},
					},
				},
			},
		}
		return customDeployment
	}

	for _, container := range customDeployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			container.StartupProbe.FailureThreshold = upgradeFailureThreshold
			return customDeployment
		}
	}

	customDeployment.Spec.Template.Spec.Containers = append(customDeployment.Spec.Template.Spec.Containers, container)
	return customDeployment
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

func registerUpgradedDoguVersion(cesreg doguRegistrator, toDogu *core.Dogu) error {
	err := cesreg.RegisterDoguVersion(toDogu)
	if err != nil {
		return fmt.Errorf("failed to register upgrade: %w", err)
	}

	return nil
}

func registerNewServiceAccount(ctx context.Context, saCreator serviceAccountCreator, resource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := saCreator.CreateAll(ctx, resource.Namespace, toDogu)
	if err != nil {
		if err != nil {
			return fmt.Errorf("failed to register service accounts: %w", err)
		}
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

func (ue *upgradeExecutor) applyCustomK8sScripts(ctx context.Context, toDoguResource *k8sv1.Dogu, execPod util.ExecPod) (*appsv1.Deployment, error) {
	var customK8sResources map[string]string
	customK8sResources, err := extractCustomK8sResources(ctx, ue.k8sFileExtractor, execPod)
	if err != nil {
		return nil, err
	}

	if len(customK8sResources) > 0 {
		ue.normalEventf(toDoguResource, "Applying/Updating custom dogu resources to the cluster: [%s]", util.GetMapKeysAsString(customK8sResources))
	}

	return applyCustomK8sResources(ctx, ue.collectApplier, toDoguResource, customK8sResources)
}

func extractCustomK8sResources(ctx context.Context, extractor fileExtractor, execPod util.ExecPod) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, execPod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier collectApplier, toDoguResource *k8sv1.Dogu, customK8sResources map[string]string) (*appsv1.Deployment, error) {
	resources, err := collectApplier.CollectApply(ctx, customK8sResources, toDoguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return resources, nil
}

func (ue *upgradeExecutor) applyPreUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu, execPod util.ExecPod) error {
	preUpgradeScriptCmd := toDogu.GetExposedCommand(core.ExposedCommandPreUpgrade)
	if preUpgradeScriptCmd == nil {
		return nil
	}

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

func copyPreUpgradeScriptFromPodToPod(ctx context.Context, execPod util.ExecPod, preUpgradeScriptCmd *core.ExposedCommand) error {
	// the exec pod is based on the image-to-be-upgraded. Thus, upgrade scripts should have the right executable
	// permissions which should be retained during the file copy.
	const copyCmd = "/bin/cp"
	// since we copy to a directory we can here ignore any included directories in the command
	// example: copy from /resource/myscript.sh to /dogu-reserved/myscript.sh
	copyPreUpgradeScriptCmd := resource.NewShellCommand(copyCmd, preUpgradeScriptCmd.Command, resource.DoguReservedPath)

	out, err := execPod.Exec(ctx, copyPreUpgradeScriptCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s' in execpod, stdout: '%s':  %w", copyPreUpgradeScriptCmd.String(), out, err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPreUpgradeScriptToOlderDogu(ctx context.Context, fromDogu *core.Dogu, toDoguResource *k8sv1.Dogu, preUpgradeCmd *core.ExposedCommand) error {
	// possibly create the necessary directory structure for the following copy action
	err := ue.createMissingUpgradeDirs(ctx, toDoguResource, preUpgradeCmd)
	if err != nil {
		return err
	}

	err = ue.copyPreUpgradeScriptToDoguReserved(ctx, toDoguResource, preUpgradeCmd)
	if err != nil {
		return err
	}

	return ue.executePreUpgradeScript(ctx, fromDogu, toDoguResource, preUpgradeCmd)
}

func (ue *upgradeExecutor) copyPreUpgradeScriptToDoguReserved(ctx context.Context, toDoguResource *k8sv1.Dogu, preUpgradeCmd *core.ExposedCommand) error {
	preUpgradeBackCopyCmd := getPreUpgradeBackCopyCmd(preUpgradeCmd)

	// the name of the dogu being upgrade must not change over the course of an installation.
	nameOfTheDoguBeingUpgraded := toDoguResource.Name
	// copy the file back to the directory where it resided in the upgrade image
	// example: /dogu-reserved/myscript.sh to /resource/myscript.sh
	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, nameOfTheDoguBeingUpgraded, toDoguResource.Namespace, preUpgradeBackCopyCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeBackCopyCmd, outBuf, err)
	}

	return nil
}

func getPreUpgradeBackCopyCmd(preUpgradeCmd *core.ExposedCommand) *resource.ShellCommand {
	_, fileName := filepath.Split(preUpgradeCmd.Command)
	filePathInDoguReserved := filepath.Join(resource.DoguReservedPath, fileName)

	return resource.NewShellCommand("/bin/cp", filePathInDoguReserved, preUpgradeCmd.Command)
}

func (ue *upgradeExecutor) createMissingUpgradeDirs(ctx context.Context, toDoguResource *k8sv1.Dogu, cmd *core.ExposedCommand) error {
	baseDir, _ := filepath.Split(cmd.Command)

	mkdirCmd := resource.NewShellCommand("/bin/mkdir", "-p", baseDir)

	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, toDoguResource.Name, toDoguResource.Namespace, mkdirCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", mkdirCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) executePreUpgradeScript(ctx context.Context, fromDogu *core.Dogu, toDoguResource *k8sv1.Dogu, cmd *core.ExposedCommand) error {
	// the previous steps took great lengths to copy the pre-upgrade script to the original place so we can finally
	// execute it as described by the dogu.json.
	preUpgradeCmd := resource.NewShellCommand(cmd.Command, fromDogu.Version, toDoguResource.Spec.Version)

	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, toDoguResource.Name, toDoguResource.Namespace, preUpgradeCmd)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", preUpgradeCmd, outBuf, err)
	}

	return nil
}

func (ue *upgradeExecutor) applyPostUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error {
	postUpgradeCmd := toDogu.GetExposedCommand(core.ExposedCommandPostUpgrade)
	if postUpgradeCmd == nil {
		return nil
	}

	ue.normalEventf(toDoguResource, "Applying optional post-upgrade scripts...")
	return ue.executePostUpgradeScript(ctx, toDoguResource, fromDogu, postUpgradeCmd)
}

func (ue *upgradeExecutor) executePostUpgradeScript(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *core.Dogu, postUpgradeCmd *core.ExposedCommand) error {
	postUpgradeShellCmd := resource.NewShellCommand(postUpgradeCmd.Command, fromDogu.Version, toDoguResource.Spec.Version)
	outBuf, err := ue.doguCommandExecutor.ExecCommandForDogu(ctx, toDoguResource.Name, toDoguResource.Namespace, postUpgradeShellCmd, resource.ContainersStarted)
	if err != nil {
		return fmt.Errorf("failed to execute '%s': output: '%s': %w", postUpgradeShellCmd, outBuf, err)
	}

	return nil
}

func updateDoguResources(ctx context.Context, upserter resourceUpserter, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error {
	err := upserter.ApplyDoguResource(ctx, toDoguResource, toDogu, image, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to update dogu resources: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) normalEventf(doguResource *k8sv1.Dogu, msg string, args ...interface{}) {
	ue.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, msg, args...)
}

func deleteExecPod(ctx context.Context, execPod util.ExecPod, recorder record.EventRecorder, doguResource *k8sv1.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, EventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
