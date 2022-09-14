package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-apply-lib/apply"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"

	"github.com/go-logr/logr"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type imageRegistry interface {
	// PullImageConfig pulls a given container image by name.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

type applier interface {
	// ApplyWithOwner applies a K8s resource as YAML doc.
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error)
}

type serviceAccountCreator interface {
	// CreateAll creates K8s services accounts for a dogu
	CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error
}

// doguRegistrator is used to register dogus
type doguRegistrator interface {
	RegisterDoguVersion(dogu *core.Dogu) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}

type resourceUpserter interface {
	// ApplyDoguResource generates K8s resources from a given dogu and creates/updates them in the cluster.
	ApplyDoguResource(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error
}

type upgradeExecutor struct {
	client                client.Client
	imageRegistry         imageRegistry
	collectApplier        collectApplier
	fileExtractor         fileExtractor
	serviceAccountCreator serviceAccountCreator
	doguRegistrator       doguRegistrator
	resourceUpserter      resourceUpserter
}

func NewUpgradeExecutor(client client.Client, imageRegistry imageRegistry, collectApplier collectApplier, fileExtractor fileExtractor, serviceAccountCreator serviceAccountCreator, registry registry.Registry) *upgradeExecutor {

	doguRegistrator := cesregistry.NewCESDoguRegistrator(client, registry, nil)
	limitPatcher := limit.NewDoguDeploymentLimitPatcher(registry)
	upserter := resource.NewUpserter(client, limitPatcher)

	return &upgradeExecutor{
		client:                client,
		imageRegistry:         imageRegistry,
		collectApplier:        collectApplier,
		fileExtractor:         fileExtractor,
		serviceAccountCreator: serviceAccountCreator,
		doguRegistrator:       doguRegistrator,
		resourceUpserter:      upserter,
	}
}

func (ue *upgradeExecutor) Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := toDoguResource.ChangeState(ctx, ue.client, k8sv1.DoguStatusUpgrading)
	if err != nil {
		return err
	}

	err = registerUpgradedDoguVersion(ue.doguRegistrator, toDogu)
	if err != nil {
		return err
	}

	err = registerNewServiceAccount(ctx, ue.serviceAccountCreator, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	imageConfigFile, err := pullUpgradeImage(ctx, ue.imageRegistry, toDogu)
	if err != nil {
		return err
	}

	var customK8sResources map[string]string
	customK8sResources, err = extractCustomK8sResources(ctx, ue.fileExtractor, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	customDeployment, err := applyCustomK8sResources(ctx, ue.collectApplier, toDoguResource, customK8sResources)
	if err != nil {
		return err
	}

	err = updateDoguResources(ctx, ue.resourceUpserter, toDoguResource, toDogu, imageConfigFile, customDeployment)
	if err != nil {
		return err
	}

	err = toDoguResource.ChangeState(ctx, ue.client, k8sv1.DoguStatusInstalled)
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

func extractCustomK8sResources(ctx context.Context, extractor fileExtractor, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) (map[string]string, error) {
	resources, err := extractor.ExtractK8sResourcesFromContainer(ctx, toDoguResource, toDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to extract custom K8s resources: %w", err)
	}

	return resources, nil
}

func applyCustomK8sResources(ctx context.Context, collectApplier collectApplier, toDoguResource *k8sv1.Dogu, customK8sResources map[string]string) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)
	resources, err := collectApplier.CollectApply(logger, customK8sResources, toDoguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return resources, nil
}

func updateDoguResources(ctx context.Context, upserter resourceUpserter, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error {
	err := upserter.ApplyDoguResource(ctx, toDoguResource, toDogu, image, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to apply custom K8s resources: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) createServiceAccounts(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := ue.serviceAccountCreator.CreateAll(ctx, toDoguResource.Namespace, toDogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) applyCustomK8sResources(customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	if len(customK8sResources) == 0 {
		return nil, nil
	}

	return nil, nil
}

func (ue *upgradeExecutor) handleCustomK8sResources(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}
