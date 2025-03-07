package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/v3/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
)

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

var clientSetGetter = func(c *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(c)
}

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	scheme                    *runtime.Scheme
	installManager            installManager
	upgradeManager            upgradeManager
	deleteManager             deleteManager
	volumeManager             volumeManager
	ingressAnnotationsManager additionalIngressAnnotationsManager
	supportManager            supportManager
	exportManager             exportManager
	startStopManager          DoguStartStopManager
	securityContextManager    securityContextManager
	recorder                  record.EventRecorder
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, ecosystemClient ecoSystem.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder) (*DoguManager, *util.ManagerSet, error) {
	ctx := context.Background()
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, nil, err
	}

	clientSet, err := clientSetGetter(restConfig)
	if err != nil {
		return nil, nil, err
	}

	configRepos := createConfigRepositories(clientSet, operatorConfig.Namespace)
	// At this point, the operator's client is only ready AFTER the operator's Start(...) was called.
	// Instead we must use our own client to avoid an immediate cache error: "the cache is not started, can not read objects"
	mgrSet, err := createMgrSet(ctx, restConfig, client, clientSet, ecosystemClient, operatorConfig, configRepos)
	if err != nil {
		return nil, nil, err
	}

	installManager := NewDoguInstallManager(client, mgrSet, eventRecorder, configRepos)

	upgradeManager := NewDoguUpgradeManager(client, mgrSet, eventRecorder)

	deleteManager := NewDoguDeleteManager(client, operatorConfig, mgrSet, eventRecorder, configRepos)

	supportManager := NewDoguSupportManager(client, mgrSet, eventRecorder)

	exportManager := NewDoguExportManager(client, mgrSet, eventRecorder)

	volumeManager := NewDoguVolumeManager(client, eventRecorder)

	ingressAnnotationsManager := NewDoguAdditionalIngressAnnotationsManager(client, eventRecorder)

	securityContextManager := NewDoguSecurityContextManager(mgrSet, eventRecorder)

	startStopManager := newDoguStartStopManager(ecosystemClient.Dogus(operatorConfig.Namespace), clientSet.AppsV1().Deployments(operatorConfig.Namespace), clientSet.CoreV1().Pods(operatorConfig.Namespace))

	return &DoguManager{
		scheme:                    client.Scheme(),
		installManager:            installManager,
		upgradeManager:            upgradeManager,
		deleteManager:             deleteManager,
		supportManager:            supportManager,
		exportManager:             exportManager,
		volumeManager:             volumeManager,
		ingressAnnotationsManager: ingressAnnotationsManager,
		startStopManager:          startStopManager,
		securityContextManager:    securityContextManager,
		recorder:                  eventRecorder,
	}, mgrSet, nil
}

func createMgrSet(ctx context.Context, restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient ecoSystem.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, configRepos util.ConfigRepositories) (*util.ManagerSet, error) {
	imageGetter := newAdditionalImageGetter(clientSet, operatorConfig.Namespace)
	additionalImageChownInitContainer, err := imageGetter.imageForKey(ctx, config.ChownInitImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}
	additionalImages := map[string]string{config.ChownInitImageConfigmapNameKey: additionalImageChownInitContainer}

	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = k8sv2.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}

	mgrSet, err := util.NewManagerSet(restConfig, client, clientSet, ecosystemClient, operatorConfig, configRepos, applier, additionalImages)
	if err != nil {
		return nil, fmt.Errorf("could not create manager set: %w", err)
	}
	return mgrSet, err
}

// Install installs a dogu resource.
func (m *DoguManager) Install(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")
	return m.installManager.Install(ctx, doguResource)
}

// Upgrade upgrades a dogu resource.
func (m *DoguManager) Upgrade(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, upgrade.EventReason, "Starting upgrade...")
	return m.upgradeManager.Upgrade(ctx, doguResource)
}

// Delete deletes a dogu resource.
func (m *DoguManager) Delete(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, DeinstallEventReason, "Starting deinstallation...")
	return m.deleteManager.Delete(ctx, doguResource)
}

// SetDoguDataVolumeSize sets the dataVolumeSize from the dogu resource to the data PVC from the dogu.
func (m *DoguManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Start volume expansion...")
	return m.volumeManager.SetDoguDataVolumeSize(ctx, doguResource)
}

// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
func (m *DoguManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, AdditionalIngressAnnotationsChangeEventReason, "Start additional ingress annotations change...")
	return m.ingressAnnotationsManager.SetDoguAdditionalIngressAnnotations(ctx, doguResource)
}

// UpdateDeploymentWithSecurityContext edits the securityContext of the deployment
func (m *DoguManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Start security context change...")
	return m.securityContextManager.UpdateDeploymentWithSecurityContext(ctx, doguResource)
}

// StartDogu scales a stopped dogu to 1.
func (m *DoguManager) StartDogu(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, StartDoguEventReason, "Starting dogu...")
	return m.startStopManager.StartDogu(ctx, doguResource)
}

// StopDogu scales a running dogu to 0.
func (m *DoguManager) StopDogu(ctx context.Context, doguResource *k8sv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, StopDoguEventReason, "Stopping dogu...")
	return m.startStopManager.StopDogu(ctx, doguResource)
}

func (m *DoguManager) CheckStarted(ctx context.Context, doguResource *k8sv2.Dogu) error {
	err := m.startStopManager.CheckStarted(ctx, doguResource)
	if err == nil {
		m.recorder.Event(doguResource, corev1.EventTypeNormal, StartDoguEventReason, "Dogu started.")
	}

	return err
}

func (m *DoguManager) CheckStopped(ctx context.Context, doguResource *k8sv2.Dogu) error {
	err := m.startStopManager.CheckStopped(ctx, doguResource)
	if err == nil {
		m.recorder.Event(doguResource, corev1.EventTypeNormal, StopDoguEventReason, "Dogu stopped.")
	}

	return err
}

// HandleSupportMode handles the support flag in the dogu spec.
func (m *DoguManager) HandleSupportMode(ctx context.Context, doguResource *k8sv2.Dogu) (bool, error) {
	return m.supportManager.HandleSupportMode(ctx, doguResource)
}

func (m *DoguManager) HandleExportMode(ctx context.Context, doguResource *k8sv2.Dogu) (bool, error) {
	return m.exportManager.HandleExportMode(ctx, doguResource)
}

// createConfigRepositories creates the repositories for global, dogu and sensitive dogu configs that are based on
// k8s resources (configmaps / secrets)
func createConfigRepositories(clientSet kubernetes.Interface, namespace string) util.ConfigRepositories {
	configMapClient := clientSet.CoreV1().ConfigMaps(namespace)
	secretsClient := clientSet.CoreV1().Secrets(namespace)

	return util.ConfigRepositories{
		GlobalConfigRepository:  repository.NewGlobalConfigRepository(configMapClient),
		DoguConfigRepository:    repository.NewDoguConfigRepository(configMapClient),
		SensitiveDoguRepository: repository.NewSensitiveDoguConfigRepository(secretsClient),
	}
}
