package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-apply-lib/apply"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	scheme                    *runtime.Scheme
	installManager            cloudogu.InstallManager
	upgradeManager            cloudogu.UpgradeManager
	deleteManager             cloudogu.DeleteManager
	volumeManager             cloudogu.VolumeManager
	ingressAnnotationsManager cloudogu.AdditionalIngressAnnotationsManager
	supportManager            cloudogu.SupportManager
	recorder                  record.EventRecorder
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, eventRecorder record.EventRecorder) (*DoguManager, error) {
	err := validateKeyProvider(cesRegistry.GlobalConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to validate key provider: %w", err)
	}

	ctx := context.Background()
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("failed to find controller REST config: %w", err)
		}
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)

	// At this point, the operator's client is only ready AFTER the operator's Start(...) was called.
	// Instead we must use our own client to avoid an immediate cache error: "the cache is not started, can not read objects"
	imageGetter := newAdditionalImageGetter(clientSet, operatorConfig.Namespace)
	additionalImageChownInitContainer, err := imageGetter.ImageForKey(ctx, config.ChownInitImageConfigmapNameKey)
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
	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}

	mgrSet, err := util.NewManagerSet(restConfig, client, clientSet, operatorConfig, cesRegistry, applier, additionalImages)
	if err != nil {
		return nil, fmt.Errorf("could not create manager set: %w", err)
	}

	installManager := NewDoguInstallManager(client, operatorConfig, cesRegistry, mgrSet, eventRecorder)
	if err != nil {
		return nil, err
	}

	upgradeManager := NewDoguUpgradeManager(client, operatorConfig, cesRegistry, mgrSet, eventRecorder)
	if err != nil {
		return nil, err
	}

	deleteManager := NewDoguDeleteManager(client, operatorConfig, cesRegistry, mgrSet, eventRecorder)
	if err != nil {
		return nil, err
	}

	supportManager, _ := NewDoguSupportManager(client, operatorConfig, cesRegistry, mgrSet, eventRecorder)

	volumeManager := NewDoguVolumeManager(client, eventRecorder)

	ingressAnnotationsManager := NewDoguAdditionalIngressAnnotationsManager(client, eventRecorder)

	return &DoguManager{
		scheme:                    client.Scheme(),
		installManager:            installManager,
		upgradeManager:            upgradeManager,
		deleteManager:             deleteManager,
		supportManager:            supportManager,
		volumeManager:             volumeManager,
		ingressAnnotationsManager: ingressAnnotationsManager,
		recorder:                  eventRecorder,
	}, nil
}

func validateKeyProvider(globalConfig cesregistry.ConfigurationContext) error {
	exists, err := globalConfig.Exists("key_provider")
	if err != nil {
		return fmt.Errorf("failed to query key provider: %w", err)
	}

	if !exists {
		err = globalConfig.Set("key_provider", "pkcs1v15")
		if err != nil {
			return fmt.Errorf("failed to set default key provider: %w", err)
		}
		log.Log.Info("No key provider found. Use default pkcs1v15.")
	}

	return nil
}

// Install installs a dogu resource.
func (m *DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")
	return m.installManager.Install(ctx, doguResource)
}

// Upgrade upgrades a dogu resource.
func (m *DoguManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, upgrade.EventReason, "Starting upgrade...")
	return m.upgradeManager.Upgrade(ctx, doguResource)
}

// Delete deletes a dogu resource.
func (m *DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, DeinstallEventReason, "Starting deinstallation...")
	return m.deleteManager.Delete(ctx, doguResource)
}

// SetDoguDataVolumeSize sets the dataVolumeSize from the dogu resource to the data PVC from the dogu.
func (m *DoguManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Start volume expansion...")
	return m.volumeManager.SetDoguDataVolumeSize(ctx, doguResource)
}

// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
func (m *DoguManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, AdditionalIngressAnnotationsChangeEventReason, "Start additional ingress annotations change...")
	return m.ingressAnnotationsManager.SetDoguAdditionalIngressAnnotations(ctx, doguResource)
}

// HandleSupportMode handles the support flag in the dogu spec.
func (m *DoguManager) HandleSupportMode(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	return m.supportManager.HandleSupportMode(ctx, doguResource)
}
