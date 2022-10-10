package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EventReason                     = "Upgrading"
	ErrorOnFailedUpgradeEventReason = "ErrUpgrade"
)

const (
	exposedCommandPreUpgrade = "pre-upgrade"
)

type imageRegistry interface {
	// PullImageConfig pulls a given container image by name.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type k8sFileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error)
}

type upgradeScriptFileExtractor interface {
	// ExtractScriptResourcesFromContainer extracts a script from a dogu image and returns them in a map filename->content.
	ExtractScriptResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, exposedCommandFilter string) (map[string]string, error)
}

type serviceAccountCreator interface {
	// CreateAll creates K8s services accounts for a dogu
	CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error
}

type doguRegistrator interface {
	// RegisterDoguVersion registers a certain dogu in a CES instance.
	RegisterDoguVersion(dogu *core.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}

type resourceUpserter interface {
	// ApplyDoguResource generates K8s resources from a given dogu and creates/updates them in the cluster.
	ApplyDoguResource(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error
}

type upgradeExecutor struct {
	client                     client.Client
	imageRegistry              imageRegistry
	collectApplier             collectApplier
	k8sFileExtractor           k8sFileExtractor
	upgradeScriptFileExtractor upgradeScriptFileExtractor
	serviceAccountCreator      serviceAccountCreator
	doguRegistrator            doguRegistrator
	resourceUpserter           resourceUpserter
	eventRecorder              record.EventRecorder
}

// NewUpgradeExecutor creates a new upgrade executor.
func NewUpgradeExecutor(
	client client.Client,
	imageRegistry imageRegistry,
	collectApplier collectApplier,
	k8sFileExtractor k8sFileExtractor,
	upgradeScriptFileExtractor upgradeScriptFileExtractor,
	serviceAccountCreator serviceAccountCreator,
	registry registry.Registry,
	eventRecorder record.EventRecorder,
) *upgradeExecutor {
	doguRegistrator := cesregistry.NewCESDoguRegistrator(client, registry, nil)
	limitPatcher := limit.NewDoguDeploymentLimitPatcher(registry)
	upserter := resource.NewUpserter(client, limitPatcher)

	return &upgradeExecutor{
		client:                     client,
		imageRegistry:              imageRegistry,
		collectApplier:             collectApplier,
		k8sFileExtractor:           k8sFileExtractor,
		upgradeScriptFileExtractor: upgradeScriptFileExtractor,
		serviceAccountCreator:      serviceAccountCreator,
		doguRegistrator:            doguRegistrator,
		resourceUpserter:           upserter,
		eventRecorder:              eventRecorder,
	}
}

// Upgrade executes all necessary steps to update a dogu to a new version.
func (ue *upgradeExecutor) Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
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
	var customK8sResources map[string]string
	customK8sResources, err = extractCustomK8sResources(ctx, ue.k8sFileExtractor, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Extracting optional upgrade scripts...")

	upgradeScripts, err := extractUpgradeScripts(ctx, ue.upgradeScriptFileExtractor, toDoguResource, toDogu)
	if err != nil {
		return err
	}
	if upgradeScripts != nil {
		// todo delete me
	}

	if len(customK8sResources) > 0 {
		ue.normalEventf(toDoguResource, "Applying/Updating custom dogu resources to the cluster: [%s]", util.GetMapKeysAsString(customK8sResources))
	}
	customDeployment, err := applyCustomK8sResources(ctx, ue.collectApplier, toDoguResource, customK8sResources)
	if err != nil {
		return err
	}

	ue.normalEventf(toDoguResource, "Updating dogu resources in the cluster...")
	err = updateDoguResources(ctx, ue.resourceUpserter, toDoguResource, toDogu, imageConfigFile, customDeployment)
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

func extractCustomK8sResources(ctx context.Context, extractor k8sFileExtractor, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, toDoguResource, toDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func extractUpgradeScripts(ctx context.Context, extractor upgradeScriptFileExtractor, doguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	return nil, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier collectApplier, toDoguResource *k8sv1.Dogu, customK8sResources map[string]string) (*appsv1.Deployment, error) {
	resources, err := collectApplier.CollectApply(ctx, customK8sResources, toDoguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return resources, nil
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
