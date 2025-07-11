package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-apply-lib/apply"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
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
	startStopManager          startStopManager
	securityContextManager    securityContextManager
	additionalMountsManager   additionalMountsManager
	recorder                  record.EventRecorder
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder) (*DoguManager, error) {
	ctx := context.Background()
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := clientSetGetter(restConfig)
	if err != nil {
		return nil, err
	}

	configRepos := createConfigRepositories(clientSet, operatorConfig.Namespace)
	// At this point, the operator's client is only ready AFTER the operator's Start(...) was called.
	// Instead we must use our own client to avoid an immediate cache error: "the cache is not started, can not read objects"
	mgrSet, err := createMgrSet(ctx, restConfig, client, clientSet, ecosystemClient, operatorConfig, configRepos)
	if err != nil {
		return nil, err
	}

	installManager := NewDoguInstallManager(client, mgrSet, eventRecorder, configRepos)

	upgradeManager := NewDoguUpgradeManager(client, mgrSet, eventRecorder)

	deleteManager := NewDoguDeleteManager(client, operatorConfig, mgrSet, eventRecorder, configRepos)

	supportManager := NewDoguSupportManager(client, mgrSet, eventRecorder)

	doguInterface := ecosystemClient.Dogus(operatorConfig.Namespace)
	exportManager := NewDoguExportManager(
		doguInterface,
		clientSet.CoreV1().Pods(operatorConfig.Namespace),
		clientSet.AppsV1().Deployments(operatorConfig.Namespace),
		mgrSet.ResourceUpserter,
		mgrSet.LocalDoguFetcher,
		eventRecorder,
	)

	volumeManager := NewDoguVolumeManager(client, eventRecorder, doguInterface)

	ingressAnnotationsManager := NewDoguAdditionalIngressAnnotationsManager(client, eventRecorder)

	securityContextManager := NewDoguSecurityContextManager(mgrSet, eventRecorder)

	startStopManager := newDoguStartStopManager(
		mgrSet.ResourceUpserter,
		mgrSet.LocalDoguFetcher,
		ecosystemClient.Dogus(operatorConfig.Namespace),
		clientSet.AppsV1().Deployments(operatorConfig.Namespace),
	)

	additionalMountsManager := NewDoguAdditionalMountManager(clientSet.AppsV1().Deployments(operatorConfig.Namespace), mgrSet, doguInterface)

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
		additionalMountsManager:   additionalMountsManager,
		recorder:                  eventRecorder,
	}, nil
}

func createMgrSet(ctx context.Context, restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, configRepos util.ConfigRepositories) (*util.ManagerSet, error) {
	imageGetter := newAdditionalImageGetter(clientSet, operatorConfig.Namespace)
	additionalImageChownInitContainer, err := imageGetter.imageForKey(ctx, config.ChownInitImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalExportModeContainer, err := imageGetter.imageForKey(ctx, config.ExporterImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalMountsContainer, err := imageGetter.imageForKey(ctx, config.AdditionalMountsInitContainerImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalImages := map[string]string{config.ChownInitImageConfigmapNameKey: additionalImageChownInitContainer,
		config.ExporterImageConfigmapNameKey:                      additionalExportModeContainer,
		config.AdditionalMountsInitContainerImageConfigmapNameKey: additionalMountsContainer}

	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = doguv2.AddToScheme(scheme)
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
func (m *DoguManager) Install(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")
	return m.installManager.Install(ctx, doguResource)
}

// Upgrade upgrades a dogu resource.
func (m *DoguManager) Upgrade(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, upgrade.EventReason, "Starting upgrade...")
	return m.upgradeManager.Upgrade(ctx, doguResource)
}

// Delete deletes a dogu resource.
func (m *DoguManager) Delete(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, DeinstallEventReason, "Starting deinstallation...")
	return m.deleteManager.Delete(ctx, doguResource)
}

// SetDoguDataVolumeSize sets the dataVolumeSize from the dogu resource to the data PVC from the dogu.
func (m *DoguManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Start volume expansion...")
	return m.volumeManager.SetDoguDataVolumeSize(ctx, doguResource)
}

// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
func (m *DoguManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, AdditionalIngressAnnotationsChangeEventReason, "Start additional ingress annotations change...")
	return m.ingressAnnotationsManager.SetDoguAdditionalIngressAnnotations(ctx, doguResource)
}

// UpdateDeploymentWithSecurityContext edits the securityContext of the deployment
func (m *DoguManager) UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, SecurityContextChangeEventReason, "Start security context change...")
	return m.securityContextManager.UpdateDeploymentWithSecurityContext(ctx, doguResource)
}

// StartStopDogu starts or stops the dogu.
func (m *DoguManager) StartStopDogu(ctx context.Context, doguResource *doguv2.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, StartStopDoguEventReason, "Starting/Stopping dogu...")
	return m.startStopManager.StartStopDogu(ctx, doguResource)
}

// UpdateExportMode activates/deactivates the export mode for the dogu
func (m *DoguManager) UpdateExportMode(ctx context.Context, doguResource *doguv2.Dogu) error {
	err := m.exportManager.UpdateExportMode(ctx, doguResource)
	if err == nil {
		m.recorder.Event(doguResource, corev1.EventTypeNormal, ChangeExportModeEventReason, "export-mode changing...")
	}

	return err
}

// HandleSupportMode handles the support flag in the dogu spec.
func (m *DoguManager) HandleSupportMode(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	return m.supportManager.HandleSupportMode(ctx, doguResource)
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

func (m *DoguManager) AdditionalMountsChanged(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	return m.additionalMountsManager.AdditionalMountsChanged(ctx, doguResource)
}

func (m *DoguManager) UpdateAdditionalMounts(ctx context.Context, doguResource *doguv2.Dogu) error {
	return m.additionalMountsManager.UpdateAdditionalMounts(ctx, doguResource)
}
