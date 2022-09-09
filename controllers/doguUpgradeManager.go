package controllers

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-apply-lib/apply"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/controllers/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type premisesChecker interface {
	Check(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *core.Dogu, toDogu *core.Dogu) error
}

type doguFetcher interface {
	Fetch(doguResource *k8sv1.Dogu) (fromDogu *core.Dogu, toDogu *core.Dogu, err error)
}

type upgradeabilityChecker interface {
	Check(fromDogu *core.Dogu, toDogu *core.Dogu, forceUpgrade bool) error
}

// NewDoguUpgradeManager creates a new instance of doguUpgradeManager which handles dogu upgrades.
func NewDoguUpgradeManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry,
	eventRecorder record.EventRecorder) (*doguUpgradeManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	imageRegistry := registry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}

	fileExtract := newPodFileExtractor(client, restConfig, clientSet)

	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add applier scheme to dogu CRD scheme handling: %w", err)
	}

	doguLocalRegistry := cesRegistry.DoguRegistry()

	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)

	doguFetcher := upgrade.NewDoguFetcher(doguLocalRegistry, doguRemoteRegistry)

	depValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, doguLocalRegistry)
	doguChecker := health.NewDoguChecker(client, doguLocalRegistry)
	premisesChecker := upgrade.NewPremisesChecker(depValidator, doguChecker, doguChecker)

	return &doguUpgradeManager{
		client:                client,
		scheme:                scheme,
		eventRecorder:         eventRecorder,
		imageRegistry:         imageRegistry,
		serviceAccountCreator: serviceAccountCreator,
		applier:               applier,
		fileExtractor:         fileExtract,
		doguFetcher:           doguFetcher,
		premisesChecker:       premisesChecker,
		upgradeabilityChecker: upgrade.NewUpgradeabilityChecker(),
	}, nil
}

type doguUpgradeManager struct {
	client                client.Client
	scheme                *runtime.Scheme
	eventRecorder         record.EventRecorder
	doguLocalRegistry     cesregistry.DoguRegistry
	doguRemoteRegistry    cesremote.Registry
	imageRegistry         imageRegistry
	doguRegistrator       doguRegistrator
	serviceAccountCreator serviceAccountCreator
	applier               applier
	fileExtractor         fileExtractor
	premisesChecker       premisesChecker
	doguFetcher           doguFetcher
	upgradeabilityChecker upgradeabilityChecker
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	upgradeDoguName := doguResource.Spec.Name
	upgradeDoguVersion := doguResource.Spec.Version

	localDogu, remoteDogu, err := dum.doguFetcher.Fetch(doguResource)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error getting dogus for upgrade: %s", err)
		return fmt.Errorf("dogu upgrade failed: %w", err)
	}

	dum.normalEvent(doguResource, "Checking premises...")

	err = dum.premisesChecker.Check(ctx, doguResource, localDogu, remoteDogu)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedPremisesUpgradeEventReason, "Checking premises failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEvent(doguResource, "Checking upgradeability...")
	const forceUpgrade = false

	err = dum.upgradeabilityChecker.Check(localDogu, remoteDogu, forceUpgrade)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeabilityEventReason, "Checking upgradeability failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEvent(doguResource, "Checking upgradeability...")

	err = dum.runUpgradeSteps(ctx, doguResource, localDogu, remoteDogu)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error during upgrade: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	return nil
}

func (dum *doguUpgradeManager) runUpgradeSteps(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error {
	// collectPreUpgradeScript goes here

	err := dum.pullUpgradeImage(ctx, toDogu)
	if err != nil {
		return err
	}

	customDeployment, err := dum.applyDoguResource(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	err = dum.instantiateAndRegisterDogu(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	// since new dogus may define new SAs in later versions we should take care of that
	err = dum.createServiceAccounts(ctx, toDoguResource, toDogu)
	if err != nil {
		return err
	}

	err = dum.updateDoguResource(ctx, toDoguResource, toDogu, customDeployment)
	if err != nil {
		return err
	}

	// collectPostUpgradeScript goes here

	return nil
}

func (dum *doguUpgradeManager) pullUpgradeImage(ctx context.Context, toDogu *core.Dogu) error {
	_, err := dum.imageRegistry.PullImageConfig(ctx, toDogu.Image+":"+toDogu.Version)
	if err != nil {
		return nil
	}

	return nil
}

func (dum *doguUpgradeManager) applyDoguResource(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) (*appsv1.Deployment, error) {
	customK8sResources, err := dum.fileExtractor.ExtractK8sResourcesFromContainer(ctx, toDoguResource, toDogu)
	if err != nil {
		return nil, fmt.Errorf("failed to pull customK8sResources: %w", err)
	}

	customDeployment, err := dum.applyCustomK8sResources(customK8sResources, toDoguResource)
	if err != nil {
		return nil, err
	}

	return customDeployment, nil
}

func (dum *doguUpgradeManager) instantiateAndRegisterDogu(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := dum.doguRegistrator.RegisterDogu(ctx, toDoguResource, toDogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	return nil
}

func (dum *doguUpgradeManager) createServiceAccounts(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	err := dum.serviceAccountCreator.CreateAll(ctx, toDoguResource.Namespace, toDogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	return nil
}

func (dum *doguUpgradeManager) updateDoguResource(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, customDeployment *appsv1.Deployment) error {
	dum.normalEvent(toDoguResource, "Creating kubernetes resources...")
	err := dum.createVolumes(ctx, toDoguResource, toDogu)
	if err != nil {
		return fmt.Errorf("failed to create volumes for dogu %s: %w", toDogu.Name, err)
	}

	err = dum.patchDeployment(ctx, toDoguResource, toDogu, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to create deployment for dogu %s: %w", toDogu.Name, err)
	}

	toDoguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = toDoguResource.Update(ctx, dum.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	return nil
}

func (dum *doguUpgradeManager) normalEvent(doguResource *k8sv1.Dogu, msg string) {
	dum.eventRecorder.Event(doguResource, corev1.EventTypeNormal, UpgradeEventReason, msg)
}

func (dum *doguUpgradeManager) errorEventf(doguResource *k8sv1.Dogu, reason, msg string, err error) {
	dum.eventRecorder.Eventf(doguResource, corev1.EventTypeWarning, reason, msg, err.Error())
}

func (dum *doguUpgradeManager) applyCustomK8sResources(customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	if len(customK8sResources) == 0 {
		return nil, nil
	}

	targetNamespace := doguResource.ObjectMeta.Namespace

	namespaceTemplate := struct {
		Namespace string
	}{
		Namespace: targetNamespace,
	}

	dCollector := &deploymentCollector{collected: []*appsv1.Deployment{}}

	for file, yamlDocs := range customK8sResources {
		err := apply.NewBuilder(dum.applier).
			WithNamespace(targetNamespace).
			WithOwner(doguResource).
			WithTemplate(file, namespaceTemplate).
			WithCollector(dCollector).
			WithYamlResource(file, []byte(yamlDocs)).
			WithApplyFilter(&deploymentAntiFilter{}).
			ExecuteApply()

		if err != nil {
			return nil, err
		}
	}

	if len(dCollector.collected) > 1 {
		return nil, fmt.Errorf("expected exactly one Deployment but found %d - not sure how to continue", len(dCollector.collected))
	}
	if len(dCollector.collected) == 1 {
		return dCollector.collected[0], nil
	}

	return nil, nil
}

func (dum *doguUpgradeManager) createVolumes(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu) error {
	return nil
}

func (dum *doguUpgradeManager) patchDeployment(ctx context.Context, toDoguResource *k8sv1.Dogu, toDogu *core.Dogu, customDeployment *appsv1.Deployment) error {
	return nil
}
