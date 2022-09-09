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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type doguHealthChecker interface {
	// CheckWithResource returns nil if the dogu described by the resource is up and running.
	CheckWithResource(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type doguRecursiveHealthChecker interface {
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, localDogu *core.Dogu, currentK8sNamespace string) error
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

	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add applier scheme to dogu CRD scheme handling: %w", err)
	}

	doguLocalRegistry := cesRegistry.DoguRegistry()

	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, doguLocalRegistry)
	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)

	doguChecker := health.NewDoguChecker(client, doguLocalRegistry)
	doguRecursiveChecker := doguChecker

	return &doguUpgradeManager{
		client:                     client,
		scheme:                     scheme,
		doguLocalRegistry:          doguLocalRegistry,
		doguRemoteRegistry:         doguRemoteRegistry,
		imageRegistry:              imageRegistry,
		dependencyValidator:        dependencyValidator,
		serviceAccountCreator:      serviceAccountCreator,
		applier:                    applier,
		doguHealthChecker:          doguChecker,
		doguRecursiveHealthChecker: doguRecursiveChecker,
		eventRecorder:              eventRecorder,
	}, nil
}

type doguUpgradeManager struct {
	client                     client.Client
	scheme                     *runtime.Scheme
	doguLocalRegistry          cesregistry.DoguRegistry
	doguRemoteRegistry         cesremote.Registry
	imageRegistry              imageRegistry
	doguRegistrator            doguRegistrator
	dependencyValidator        dependencyValidator
	serviceAccountCreator      serviceAccountCreator
	applier                    applier
	doguHealthChecker          doguHealthChecker
	doguRecursiveHealthChecker doguRecursiveHealthChecker
	eventRecorder              record.EventRecorder
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	upgradeDoguName := doguResource.Spec.Name
	upgradeDoguVersion := doguResource.Spec.Version

	localDogu, remoteDogu, err := dum.getDogusForResource(doguResource)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error getting dogus for upgrade: %s", err)
		return fmt.Errorf("dogu upgrade failed: %w", err)
	}

	dum.normalEvent(doguResource, "Checking premises...")

	err = dum.checkPremises(ctx, doguResource, localDogu, remoteDogu)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedPremisesUpgradeEventReason, "Checking premises failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEvent(doguResource, "Checking upgradeability...")
	const forceUpgrade = false

	err = checkUpgradeability(localDogu, remoteDogu, forceUpgrade)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeabilityEventReason, "Checking upgradeability failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed a premise check: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	dum.normalEvent(doguResource, "Checking upgradeability...")

	steps, err := dum.collectUpgradeSteps()
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Collecting upgrade steps failed: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	err = dum.runUpgradeSteps(steps)
	if err != nil {
		dum.errorEventf(doguResource, ErrorOnFailedUpgradeEventReason, "Error during upgrade: %s", err)
		return fmt.Errorf("dogu upgrade %s:%s failed: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	return nil
}

func (dum *doguUpgradeManager) getDogusForResource(doguResource *k8sv1.Dogu) (localDogu *core.Dogu, remoteDogu *core.Dogu, err error) {
	upgradeDoguName := doguResource.Spec.Name
	upgradeDoguVersion := doguResource.Spec.Version

	localDogu, err = dum.getLocalDogu(doguResource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local dogu descriptor for %s:%s: %w", upgradeDoguName, upgradeDoguVersion, err)
	}

	remoteDogu, err = dum.getRemoteDogu(upgradeDoguName, upgradeDoguVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get remote dogu descriptor for %s:%s: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	return localDogu, remoteDogu, nil
}

func (dum *doguUpgradeManager) getLocalDogu(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	simpleDoguName := doguResource.Name
	dogu, err := dum.doguLocalRegistry.Get(simpleDoguName)
	if err != nil {
		return nil, fmt.Errorf("could not fetch the local descriptor for dogu %s: %w", doguResource.Spec.Name, err)
	}

	return dogu, nil
}

func (dum *doguUpgradeManager) getRemoteDogu(name, version string) (*core.Dogu, error) {
	return dum.doguRemoteRegistry.GetVersion(name, version)
}

func (dum *doguUpgradeManager) checkPremises(ctx context.Context, doguResource *k8sv1.Dogu, localDogu *core.Dogu, remoteDogu *core.Dogu) error {
	const premErrMsg = "premises check failed: %w"

	err := dum.checkDependencyDogusHealthy(ctx, doguResource, localDogu)
	if err != nil {
		return fmt.Errorf(premErrMsg, err)
	}

	err = dum.doguHealthChecker.CheckWithResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf(premErrMsg, err)
	}

	changeNamespace := doguResource.Spec.UpgradeConfig.AllowNamespaceSwitch
	err = checkDoguIdentity(localDogu, remoteDogu, changeNamespace)
	if err != nil {
		return fmt.Errorf(premErrMsg, err)
	}

	return nil
}

func (dum *doguUpgradeManager) checkDependencyDogusHealthy(ctx context.Context, doguResource *k8sv1.Dogu, localDogu *core.Dogu) error {
	err := dum.dependencyValidator.ValidateDependencies(localDogu)
	if err != nil {
		return err
	}

	return dum.doguRecursiveHealthChecker.CheckDependenciesRecursive(ctx, localDogu, doguResource.Namespace)

}

func checkUpgradeability(localDogu *core.Dogu, remoteDogu *core.Dogu, forceUpgrade bool) error {
	if forceUpgrade {
		return nil
	}

	err := checkVersionBeforeUpgrade(localDogu, remoteDogu)
	if err != nil {
		return fmt.Errorf("upgradeability check failed: %w", err)
	}

	return nil
}

func checkVersionBeforeUpgrade(localDogu *core.Dogu, remoteDogu *core.Dogu) error {
	localVersion, err := core.ParseVersion(localDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgradeability of local dogu: %w", err)
	}
	remoteVersion, err := core.ParseVersion(remoteDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgradeability of remote dogu: %w", err)
	}

	if remoteVersion.IsOlderOrEqualThan(localVersion) {
		return fmt.Errorf("remote version must be greater than local version '%s > %s'",
			remoteDogu.Version, localDogu.Version)
	}
	return nil
}

func checkDoguIdentity(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	if localDogu.GetSimpleName() != remoteDogu.GetSimpleName() {
		return fmt.Errorf("dogus must have the same name (%s=%s)", localDogu.GetSimpleName(), remoteDogu.GetSimpleName())
	}

	if !namespaceChange && localDogu.GetNamespace() != remoteDogu.GetNamespace() {
		return fmt.Errorf("dogus must have the same namespace (%s=%s)", localDogu.GetNamespace(), remoteDogu.GetNamespace())
	}

	return nil
}

type upgradeStep struct {
	action func() error
}

func (dum *doguUpgradeManager) collectUpgradeSteps() ([]upgradeStep, error) {
	return nil, nil
}

func (dum *doguUpgradeManager) runUpgradeSteps(steps []upgradeStep) error {
	return nil
}

func (dum *doguUpgradeManager) normalEvent(doguResource *k8sv1.Dogu, msg string) {
	dum.eventRecorder.Event(doguResource, corev1.EventTypeNormal, UpgradeEventReason, msg)
}

func (dum *doguUpgradeManager) errorEventf(doguResource *k8sv1.Dogu, reason, msg string, err error) {
	dum.eventRecorder.Eventf(doguResource, corev1.EventTypeWarning, reason, msg, err.Error())
}
