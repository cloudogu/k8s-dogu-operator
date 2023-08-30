package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	cesreg "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-host-change/pkg/alias"
	"k8s.io/client-go/rest"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
)

// NewDoguUpgradeManager creates a new instance of doguUpgradeManager which handles dogu upgrades.
func NewDoguUpgradeManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesreg.Registry,
	eventRecorder record.EventRecorder) (*doguUpgradeManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	imageRegistry := imageregistry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)

	restConfig, collectApplier, fileExtractor, executor, serviceAccountCreator, df, rdf, upserter, _, err := initManagerObjects(client, operatorConfig, cesRegistry, doguRemoteRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dogu upgrade manager objects: %w", err)
	}

	depValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())
	doguChecker := health.NewDoguChecker(client, df)
	premisesChecker := upgrade.NewPremisesChecker(depValidator, doguChecker, doguChecker)

	upgradeExecutor, _ := upgrade.NewUpgradeExecutor(
		client,
		restConfig,
		executor,
		eventRecorder,
		imageRegistry,
		collectApplier,
		fileExtractor,
		serviceAccountCreator,
		cesRegistry,
		upserter,
	)

	return &doguUpgradeManager{
		client:              client,
		eventRecorder:       eventRecorder,
		localDoguFetcher:    df,
		resourceDoguFetcher: rdf,
		premisesChecker:     premisesChecker,
		upgradeExecutor:     upgradeExecutor,
	}, nil
}

func initManagerObjects(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesreg.Registry, doguRemoteRegistry cesremote.Registry) (restConfig *rest.Config, collectApplier cloudogu.CollectApplier, fileExtractor cloudogu.FileExtractor, commandExcecutor cloudogu.CommandExecutor, serviceAccountCreator cloudogu.ServiceAccountCreator, localDoguFetcher cloudogu.LocalDoguFetcher, resourceDoguFetcher cloudogu.ResourceDoguFetcher, resourceUpserter cloudogu.ResourceUpserter, doguResourceGenerator cloudogu.DoguResourceGenerator, error error) {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to find controller REST config: %w", err)
		}
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to find cluster config: %w", err)
	}
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}
	collectApplier = resource.NewCollectApplier(applier)

	fileExtractor = exec.NewPodFileExtractor(client, restConfig, clientSet)

	doguLocalRegistry := cesRegistry.DoguRegistry()

	executor := exec.NewCommandExecutor(client, clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator = serviceaccount.NewCreator(cesRegistry, executor, client)

	df := cesregistry.NewLocalDoguFetcher(doguLocalRegistry)
	rdf := cesregistry.NewResourceDoguFetcher(client, doguRemoteRegistry)

	requirementsGenerator := resource.NewRequirementsGenerator(cesRegistry)
	hostAliasGenerator := alias.NewHostAliasGenerator(cesRegistry.GlobalConfig())
	additionalImageGetter := util.NewAdditionalImageGetter(clientSet.CoreV1().ConfigMaps(operatorConfig.Namespace))
	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), requirementsGenerator, hostAliasGenerator, additionalImageGetter)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("cannot create resource generator: %w", err)
	}

	upserter := resource.NewUpserter(client, resourceGenerator)
	return restConfig, collectApplier, fileExtractor, executor, serviceAccountCreator, df, rdf, upserter, resourceGenerator, nil
}

type doguUpgradeManager struct {
	// general purpose
	client        client.Client
	eventRecorder record.EventRecorder
	// upgrade business
	premisesChecker     cloudogu.PremisesChecker
	localDoguFetcher    cloudogu.LocalDoguFetcher
	resourceDoguFetcher cloudogu.ResourceDoguFetcher
	upgradeExecutor     cloudogu.UpgradeExecutor
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := doguResource.ChangeState(ctx, dum.client, k8sv1.DoguStatusUpgrading)
	if err != nil {
		return err
	}

	upgradeDoguName := doguResource.Spec.Name
	upgradeDoguVersion := doguResource.Spec.Version

	fromDogu, toDogu, developmentDoguMap, err := dum.getDogusForUpgrade(ctx, doguResource)
	if err != nil {
		return err
	}

	dum.normalEvent(doguResource, "Checking premises...")
	err = dum.premisesChecker.Check(ctx, doguResource, fromDogu, toDogu)
	if err != nil {
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEventf(doguResource, "Executing upgrade from %s to %s...", fromDogu.Version, toDogu.Version)
	err = dum.upgradeExecutor.Upgrade(ctx, doguResource, fromDogu, toDogu)
	if err != nil {
		return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	err = doguResource.ChangeState(ctx, dum.client, k8sv1.DoguStatusInstalled)
	if err != nil {
		return err
	}

	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, dum.client)
		if err != nil {
			// an error during deleting the developmentDoguMap is not critical, so we change the dogu state as installed earlier
			return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
		}
	}

	return nil
}

func (dum *doguUpgradeManager) getDogusForUpgrade(ctx context.Context, doguResource *k8sv1.Dogu) (*core.Dogu, *core.Dogu, *k8sv1.DevelopmentDoguMap, error) {
	fromDogu, err := dum.localDoguFetcher.FetchInstalled(doguResource.Name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dogu upgrade failed: %w", err)
	}

	toDogu, developmentDoguMap, err := dum.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dogu upgrade failed: %w", err)
	}

	return fromDogu, toDogu, developmentDoguMap, nil
}

func (dum *doguUpgradeManager) normalEvent(doguResource *k8sv1.Dogu, msg string) {
	dum.eventRecorder.Event(doguResource, corev1.EventTypeNormal, upgrade.EventReason, msg)
}

func (dum *doguUpgradeManager) normalEventf(doguResource *k8sv1.Dogu, msg string, msgArg ...interface{}) {
	dum.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, upgrade.EventReason, msg, msgArg...)
}
