package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type upgradeExecutor struct {
	client                client.Client
	imageRegistry         imageRegistry
	applier               applier
	fileExtractor         fileExtractor
	serviceAccountCreator serviceAccountCreator
}

func NewUpgradeExecutor(client client.Client, imageRegistry imageRegistry, applier applier, fileExtractor fileExtractor, serviceAccountCreator serviceAccountCreator) *upgradeExecutor {
	return &upgradeExecutor{
		client:                client,
		imageRegistry:         imageRegistry,
		applier:               applier,
		fileExtractor:         fileExtractor,
		serviceAccountCreator: serviceAccountCreator,
	}
}

func (ue *upgradeExecutor) Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error {

	err := toDoguResource.ChangeState(ctx, ue.client, k8sv1.DoguStatusUpgrading)
	if err != nil {
		return err
	}

	err = registerUpgradedDoguVersion(ctx, toDogu)
	if err != nil {
		return err
	}

	err = registerNewServiceAccount(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	imageConfigFile, err := ue.pullUpgradeImage(ctx, toDogu)
	if err != nil {
		return err
	}

	var customK8sResources map[string]string
	customK8sResources, err = extractCustomK8sResources(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	customDeployment, err := applyCustomK8sResources(ctx, toDoguResource, customK8sResources)
	if err != nil {
		return err
	}

	err = createDoguResources(ctx, toDoguResource, toDogu, imageConfigFile, customDeployment)
	if err != nil {
		return err
	}

	err = toDoguResource.ChangeState(ctx, ue.client, k8sv1.DoguStatusInstalled)
	if err != nil {
		return err
	}

	return nil
}

func registerUpgradedDoguVersion(ctx context.Context, toDogu *core.Dogu) error {
	return nil
}

func registerNewServiceAccount(ctx context.Context, resource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func (ue *upgradeExecutor) pullUpgradeImage(ctx context.Context, toDogu *core.Dogu) (*imagev1.ConfigFile, error) {
	return ue.imageRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
}

func extractCustomK8sResources(ctx context.Context, toDoguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	return nil, nil
}

func applyCustomK8sResources(ctx context.Context, toDoguResource *k8sv1.Dogu, k8sResources map[string]string) (*appsv1.Deployment, error) {
	return nil, nil
}

func createDoguResources(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, image *imagev1.ConfigFile, customDeployment *appsv1.Deployment) error {
	/*deployment*/ _, err := createDeployment(ctx, toDoguResource, customDeployment)
	if err != nil {

	}

	err = createVolumes(ctx, toDoguResource, toDogu)
	if err != nil {

	}

	err = createOrUpdateInternalServices(ctx, toDoguResource, toDogu, image)
	if err != nil {

	}

	err = createOrUpdateExternalServices(ctx, toDoguResource, toDogu)
	if err != nil {

	}

	return nil
}

func createDeployment(ctx context.Context, toDoguResource *k8sv1.Dogu, deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	return nil, nil
}

func createVolumes(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func createOrUpdateInternalServices(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, image *imagev1.ConfigFile) error {
	return nil
}

func createOrUpdateExternalServices(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func (ue *upgradeExecutor) createServiceAccounts(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := ue.serviceAccountCreator.CreateAll(ctx, toDoguResource.Namespace, toDogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) updateDoguResource(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, customDeployment *appsv1.Deployment) error {

	err := ue.patchDeployment(ctx, toDoguResource, toDogu, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to create deployment for dogu %s: %w", toDogu.Name, err)
	}

	toDoguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = toDoguResource.Update(ctx, ue.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	return nil
}

func (ue *upgradeExecutor) applyCustomK8sResources(customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	if len(customK8sResources) == 0 {
		return nil, nil
	}

	return nil, nil
}

func (ue *upgradeExecutor) createVolumes(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func (ue *upgradeExecutor) patchDeployment(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, customDeployment *appsv1.Deployment) error {
	return nil
}

func (ue *upgradeExecutor) handleCustomK8sResources(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}
