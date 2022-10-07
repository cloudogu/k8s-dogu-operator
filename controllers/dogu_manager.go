package controllers

import (
	"context"
	"fmt"

	cesregistry "github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	scheme         *runtime.Scheme
	installManager installManager
	upgradeManager upgradeManager
	deleteManager  deleteManager
	recorder       record.EventRecorder
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, eventRecorder record.EventRecorder) (*DoguManager, error) {
	err := validateKeyProvider(cesRegistry.GlobalConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to validate key provider: %w", err)
	}

	installManager, err := NewDoguInstallManager(client, operatorConfig, cesRegistry, eventRecorder)
	if err != nil {
		return nil, err
	}

	upgradeManager, err := NewDoguUpgradeManager(client, operatorConfig, cesRegistry, eventRecorder)
	if err != nil {
		return nil, err
	}

	deleteManager, err := NewDoguDeleteManager(client, cesRegistry)
	if err != nil {
		return nil, err
	}

	return &DoguManager{
		scheme:         client.Scheme(),
		installManager: installManager,
		upgradeManager: upgradeManager,
		deleteManager:  deleteManager,
		recorder:       eventRecorder,
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
