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
	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"

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

type upgradeExecutor interface {
	Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu, toDogu *core.Dogu) error
}

// NewDoguUpgradeManager creates a new instance of doguUpgradeManager which handles dogu upgrades.
func NewDoguUpgradeManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry,
	eventRecorder record.EventRecorder) (*doguUpgradeManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	imageRegistry := imageregistry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}

	fileExtractor := newPodFileExtractor(client, restConfig, clientSet)

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
	resourceGen := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))

	upgradeExecutor := upgrade.NewUpgradeExecutor(client, imageRegistry, applier, fileExtractor, serviceAccountCreator, cesRegistry, resourceGen)

	return &doguUpgradeManager{
		client:                client,
		scheme:                scheme,
		eventRecorder:         eventRecorder,
		doguFetcher:           doguFetcher,
		premisesChecker:       premisesChecker,
		upgradeabilityChecker: upgrade.NewUpgradeabilityChecker(),
		upgradeExecutor:       upgradeExecutor,
	}, nil
}

type doguUpgradeManager struct {
	// general purpose
	client        client.Client
	scheme        *runtime.Scheme
	eventRecorder record.EventRecorder
	// upgrade business
	premisesChecker       premisesChecker
	doguFetcher           doguFetcher
	upgradeabilityChecker upgradeabilityChecker
	upgradeExecutor       upgradeExecutor
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	upgradeDoguName := doguResource.Spec.Name
	upgradeDoguVersion := doguResource.Spec.Version

	// TODO feasibly fetch the toRemoteDogu from a custom dogu config map
	// config map zur lokalen Entwicklung hat gleichen Namen wie dogu
	fromLocalDogu, toRemoteDogu, err := dum.doguFetcher.Fetch(doguResource)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error getting dogus for upgrade: %s", err)
		return fmt.Errorf("dogu upgrade failed: %w", err)
	}

	dum.normalEvent(doguResource, "Checking premises...")

	err = dum.premisesChecker.Check(ctx, doguResource, fromLocalDogu, toRemoteDogu)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedPremisesUpgradeEventReason, "Checking premises failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEvent(doguResource, "Checking upgradeability...")

	err = dum.upgradeabilityChecker.Check(fromLocalDogu, toRemoteDogu, doguResource.Spec.UpgradeConfig.ForceUpgrade)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeabilityEventReason, "Checking upgradeability failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEventf(doguResource, "Executing upgrade from %s to %s...", fromLocalDogu.Version, toRemoteDogu.Version)

	err = dum.upgradeExecutor.Upgrade(ctx, doguResource, fromLocalDogu, toRemoteDogu)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error during upgrade: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	// TODO custom config map wieder löschen

	return nil
}

func (dum *doguUpgradeManager) normalEvent(doguResource *k8sv1.Dogu, msg string) {
	dum.eventRecorder.Event(doguResource, corev1.EventTypeNormal, UpgradeEventReason, msg)
}

func (dum *doguUpgradeManager) normalEventf(doguResource *k8sv1.Dogu, msg string, msgArg ...interface{}) {
	dum.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, UpgradeEventReason, msg, msgArg)
}

func (dum *doguUpgradeManager) errorEventf(doguResource *k8sv1.Dogu, reason, msg string, err error) {
	dum.eventRecorder.Eventf(doguResource, corev1.EventTypeWarning, reason, msg, err.Error())
}
