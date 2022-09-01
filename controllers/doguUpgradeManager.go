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

	currentDogu, err := dum.getCurrentDogu(doguResource)
	if err != nil {
		return err
	}

	remoteDogu, err := dum.getRemoteDogu(doguResource.Spec.Name, doguResource.Spec.Version)
	if err != nil {
		return err
	}

	err = dum.checkPremises(doguResource, currentDogu, remoteDogu)

	const forceUpgrade = false
	err = dum.checkUpgradeability(doguResource, currentDogu, remoteDogu, forceUpgrade)

	steps, err := dum.collectUpgradeSteps()

	err = dum.runUpgradeSteps(steps)
	if err != nil {
		return err
	}

	// note: there won't exist a purgeOldContainerImage step: that is the subject of Kubernetes's cluster configuration

	return nil
}

func (dum *doguUpgradeManager) getCurrentDogu(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	return nil, nil
}

func (dum *doguUpgradeManager) assertDependentDogusRunning() error {
	return nil
}

func (dum *doguUpgradeManager) assertDoguHealth() error {
	return nil
}

func (dum *doguUpgradeManager) namespaceChange() (bool, error) {
	return false, nil
}

func (dum *doguUpgradeManager) getRemoteDogu(string, string) (*core.Dogu, error) {
	return nil, nil
}

func (dum *doguUpgradeManager) assertDoguVersionChanged(namespaceChanging bool, dogu *core.Dogu) error {
	return nil
}

func (dum *doguUpgradeManager) checkPremises(doguResource *k8sv1.Dogu, dogu *core.Dogu, remoteDogu *core.Dogu) error {
	err := dum.assertDependentDogusRunning()
	if err != nil {
		return err
	}
	err = dum.assertDoguHealth()
	if err != nil {
		return err
	}
	namespaceChanging, err := dum.namespaceChange()
	if err != nil {
		return err
	}

	err = dum.assertDoguVersionChanged(namespaceChanging, remoteDogu)
	if err != nil {
		return err
	}

	return nil
}

func (dum *doguUpgradeManager) checkUpgradeability(doguResource *k8sv1.Dogu, dogu *core.Dogu, remoteDogu *core.Dogu, upgrade bool) error {
	// Upgradefähigkeit prüfen
	// wenn Force-Update dann weiter
	// wenn Dogu-lokal.Version > Dogu-remote.Version dann Fehler
	// Namensidentitätsprüfung
	// Dogu-Name
	// Namespace-Name (wenn nicht Namespace-Änderung anliegt)
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
