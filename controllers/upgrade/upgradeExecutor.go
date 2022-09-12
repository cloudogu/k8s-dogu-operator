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
	// collectPreUpgradeScript goes here

	err := ue.pullUpgradeImage(ctx, toDogu)
	if err != nil {
		return err
	}

	err = ue.handleCustomK8sResources(ctx, toDoguResource, toDogu)
	if err != nil {

	}

	customDeployment, err := ue.applyDoguResource(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	// since new dogus may define new SAs in later versions we should take care of that
	err = ue.createServiceAccounts(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	err = ue.updateDoguResource(ctx, toDoguResource, toDogu, customDeployment)
	if err != nil {
		return err
	}

	// collectPostUpgradeScript goes here

	return nil
}

func (ue *upgradeExecutor) pullUpgradeImage(ctx context.Context, toDogu *core.Dogu) error {
	_, err := ue.imageRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
	if err != nil {
		return nil
	}

	return nil
}

func (ue *upgradeExecutor) applyDoguResource(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) (*appsv1.Deployment, error) {
	customK8sResources, err := ue.fileExtractor.ExtractK8sResourcesFromContainer(ctx, toDoguResource, toDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to pull customK8sResources: %w", err)
	}

	customDeployment, err := ue.applyCustomK8sResources(customK8sResources, toDoguResource)
	if err != nil {
		return nil, err
	}

	return customDeployment, nil
}

func (ue *upgradeExecutor) updateDeployment(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {

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
	err := ue.createVolumes(ctx, toDoguResource, toDogu)
	if err != nil {
		return fmt.Errorf("failed to create volumes for dogu %s: %w", toDogu.Name, err)
	}

	err = ue.patchDeployment(ctx, toDoguResource, toDogu, customDeployment)
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

	// targetNamespace := doguResource.ObjectMeta.Namespace

	// namespaceTemplate := struct {
	// 	Namespace string
	// }{
	// 	Namespace: targetNamespace,
	// }

	// dCollector := &deploymentCollector{collected: []*appsv1.Deployment{}}
	//
	// for file, yamlDocs := range customK8sResources {
	// 	err := apply.NewBuilder(ue.applier).
	// 		WithNamespace(targetNamespace).
	// 		WithOwner(doguResource).
	// 		WithTemplate(file, namespaceTemplate).
	// 		WithCollector(dCollector).
	// 		WithYamlResource(file, []byte(yamlDocs)).
	// 		WithApplyFilter(&deploymentAntiFilter{}).
	// 		ExecuteApply()
	//
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	//
	// if len(dCollector.collected) > 1 {
	// 	return nil, fmt.Errorf("expected exactly one Deployment but found %d - not sure how to continue", len(dCollector.collected))
	// }
	// if len(dCollector.collected) == 1 {
	// 	return dCollector.collected[0], nil
	// }

	return nil, nil
}

func (ue *upgradeExecutor) createVolumes(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func (ue *upgradeExecutor) patchDeployment(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, customDeployment *appsv1.Deployment) error {
	return nil
}

func (ue *upgradeExecutor) handleCustomK8sResources(ctx context.Context, resource *k8sv1.Dogu, dogu *core.Dogu) error {
	return nil
}
