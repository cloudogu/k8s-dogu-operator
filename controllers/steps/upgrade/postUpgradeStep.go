package upgrade

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const requeueAfterRevertStartupProbe = time.Second * 3

// The PostUpgradeStep runs the post-upgrade script and reverts the startup probe to the previous value.
type PostUpgradeStep struct {
	client              k8sClient
	localDoguFetcher    localDoguFetcher
	deploymentInterface deploymentInterface
	doguCommandExecutor commandExecutor
}

func NewPostUpgradeStep(
	client client.Client,
	deploymentInterface appsv1.DeploymentInterface,
	localFetcher cesregistry.LocalDoguFetcher,
	executor exec.CommandExecutor,
) *PostUpgradeStep {
	return &PostUpgradeStep{
		client:              client,
		deploymentInterface: deploymentInterface,
		localDoguFetcher:    localFetcher,
		doguCommandExecutor: executor,
	}
}

//1. set conditions to false
//2. update deployment version
//2.1. skip if deployment version is == dogu resource spec version
//2.2. fetch local dogu json for new version
//2.3. requeue if exec pod does not exist or is not ready
//2.4. apply pre-upgrade script
//2.5. update deployment with increased startup probe failure threshold
//3. set new version as current version
//4. delete exec pod
//5. post-upgrade
//5.1. fetch local dogu json for new version
//5.2. fetch deployment
//5.3. skip if previous version is not set
//5.4. execute post-upgrade script
//5.5. revert startup probe
//5.6. requeue

func (rsps *PostUpgradeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("post-upgrade step").WithValues("dogu", doguResource.Name)

	toDogu, err := rsps.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	deployment, err := rsps.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch deployment: %w", err))
	}

	fromVersion := deployment.Annotations[previousVersionAnnotationKey]
	if fromVersion == "" ||
		rsps.startupProbeHasDefaultValue(deployment, doguResource.Name, resource.CreateStartupProbe(toDogu)) {
		return steps.Continue()
	}

	logger.Info("executing post-upgrade script")

	// Run Postupgrade Script
	err = rsps.applyPostUpgradeScript(ctx, doguResource, fromVersion, toDogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("post-upgrade failed: %w", err))
	}

	logger.Info("reverting startup probe to default value")

	// Revert probe
	err = rsps.revertStartupProbeAfterUpdate(ctx, doguResource, toDogu, deployment)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.RequeueAfter(requeueAfterRevertStartupProbe)
}

func (rsps *PostUpgradeStep) startupProbeHasDefaultValue(deployment *v1.Deployment, containerName string, probe *coreV1.Probe) bool {
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[i].StartupProbe, probe)
		}
	}
	return false
}

func (rsps *PostUpgradeStep) applyPostUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, toDogu *core.Dogu) error {
	if !toDogu.HasExposedCommand(core.ExposedCommandPostUpgrade) {
		return nil
	}

	postUpgradeCmd := toDogu.GetExposedCommand(core.ExposedCommandPostUpgrade)

	return rsps.executePostUpgradeScript(ctx, toDoguResource, fromDoguVersion, postUpgradeCmd)
}

func (rsps *PostUpgradeStep) executePostUpgradeScript(ctx context.Context, toDoguResource *v2.Dogu, fromDoguVersion string, postUpgradeCmd *core.ExposedCommand) error {
	postUpgradeShellCmd := exec.NewShellCommand(postUpgradeCmd.Command, fromDoguVersion, toDoguResource.Spec.Version)

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

func (rsps *PostUpgradeStep) revertStartupProbeAfterUpdate(ctx context.Context, toDoguResource *v2.Dogu, toDogu *core.Dogu, deployment *v1.Deployment) error {
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
