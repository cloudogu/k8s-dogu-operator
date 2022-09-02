package controllers

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/cesapp/v5/logging"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewDoguUpgradeManager creates a new instance of doguUpgradeManager which handles dogu upgrades.
func NewDoguUpgradeManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry) (*doguUpgradeManager, error) {
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

	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())
	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)

	return &doguUpgradeManager{
		client:             client,
		scheme:             scheme,
		doguRemoteRegistry: doguRemoteRegistry,
		// doguLocalRegistry:     dog,
		imageRegistry:         imageRegistry,
		doguLocalRegistry:     cesRegistry.DoguRegistry(),
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		applier:               applier,
	}, nil
}

type doguUpgradeManager struct {
	client                client.Client
	scheme                *runtime.Scheme
	doguRemoteRegistry    cesremote.Registry
	doguLocalRegistry     cesregistry.DoguRegistry
	imageRegistry         imageRegistry
	doguRegistrator       doguRegistrator
	dependencyValidator   dependencyValidator
	serviceAccountCreator serviceAccountCreator
	applier               applier
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {

	localDogu, err := dum.getLocalDogu(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get local dogu descriptor for %s:%s: %w", doguResource.Spec.Name, doguResource.Spec.Version, err)
	}

	remoteDogu, err := dum.getRemoteDogu(doguResource.Spec.Name, doguResource.Spec.Version)
	if err != nil {
		return fmt.Errorf("failed to get remote dogu descriptor for %s:%s: %w", doguResource.Spec.Name, doguResource.Spec.Version, err)
	}

	const forceUpgrade = false
	err = dum.checkPremises(doguResource, localDogu, remoteDogu)
	if err != nil {
		return fmt.Errorf("failed failed to get remote dogu descriptor for %s:%s: %w", doguResource.Spec.Name, doguResource.Spec.Version, err)
	}

	err = checkUpgradeability(localDogu, remoteDogu, forceUpgrade)
	if err != nil {

	}

	steps, err := dum.collectUpgradeSteps()

	err = dum.runUpgradeSteps(steps)
	if err != nil {
		return err
	}

	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	return nil
}

func (dum *doguUpgradeManager) getLocalDogu(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	dogu, err := dum.doguLocalRegistry.Get(doguResource.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("could not fetch the local descriptor for dogu %s: %w", doguResource.Spec.Name, err)
	}

	return dogu, nil
}

func (dum *doguUpgradeManager) checkDependentDogusRunning() error {
	return nil
}

func (dum *doguUpgradeManager) checkDoguHealth() error {
	return nil
}

func (dum *doguUpgradeManager) namespaceChange() (bool, error) {
	return false, nil
}

func (dum *doguUpgradeManager) getRemoteDogu(string, string) (*core.Dogu, error) {
	return nil, nil
}

func (dum *doguUpgradeManager) checkDoguVersionChanged(namespaceChanging bool, dogu *core.Dogu) error {
	return nil
}

func (dum *doguUpgradeManager) checkPremises(doguResource *k8sv1.Dogu, localDogu *core.Dogu, remoteDogu *core.Dogu) error {
	err := dum.checkDependentDogusRunning()
	if err != nil {
		return err
	}

	err = dum.checkDoguHealth()
	if err != nil {
		return err
	}

	namespaceChanging, err := dum.namespaceChange()
	if err != nil {
		return err
	}

	err = checkVersionBeforeUpgrade(localDogu, remoteDogu, namespaceChanging)
	if err != nil {
		return err
	}

	return nil
}

func checkUpgradeability(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	logger := logging.GetInstance()
	logger.Debugf("Check upgrade-ability of dogu versions (l:%s <-> r:%s)", localDogu.Name, localDogu.Version)

	err := checkDoguIdentity(localDogu, remoteDogu, namespaceChange)
	if err != nil {
		return fmt.Errorf("upgrade-ability check failed: %w", err)
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

func checkVersionBeforeUpgrade(localDogu *core.Dogu, remoteDogu *core.Dogu, forceUpgrade bool) error {
	if !forceUpgrade {
		return nil
	}

	localVersion, err := core.ParseVersion(localDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgrade-ability of local dogu: %w", err)
	}
	remoteVersion, err := core.ParseVersion(remoteDogu.Version)
	if err != nil {
		return fmt.Errorf("could not check upgrade-ability of remote dogu: %w", err)
	}

	if remoteVersion.IsOlderOrEqualThan(localVersion) {
		return fmt.Errorf("remote version must be greater than local version '%s > %s'",
			remoteDogu.Version, localDogu.Version)
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
