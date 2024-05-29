package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// NewDoguUpgradeManager creates a new instance of doguUpgradeManager which handles dogu upgrades.
func NewDoguUpgradeManager(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *doguUpgradeManager {
	doguChecker := health.NewDoguChecker(mgrSet.EcosystemClient, mgrSet.LocalDoguFetcher)
	premisesChecker := upgrade.NewPremisesChecker(mgrSet.DependencyValidator, doguChecker, doguChecker)

	upgradeExecutor := upgrade.NewUpgradeExecutor(client, mgrSet, eventRecorder, mgrSet.EcosystemClient)

	return &doguUpgradeManager{
		client:              client,
		ecosystemClient:     mgrSet.EcosystemClient,
		eventRecorder:       eventRecorder,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		premisesChecker:     premisesChecker,
		upgradeExecutor:     upgradeExecutor,
	}
}

type doguUpgradeManager struct {
	// general purpose
	client          client.Client
	ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface
	eventRecorder   record.EventRecorder
	// upgrade business
	premisesChecker     cloudogu.PremisesChecker
	localDoguFetcher    cloudogu.LocalDoguFetcher
	resourceDoguFetcher cloudogu.ResourceDoguFetcher
	upgradeExecutor     cloudogu.UpgradeExecutor
}

func (dum *doguUpgradeManager) Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := doguResource.ChangeStateWithRetry(ctx, dum.client, k8sv1.DoguStatusUpgrading)
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

	err = doguResource.ChangeStateWithRetry(ctx, dum.client, k8sv1.DoguStatusInstalled)
	if err != nil {
		return err
	}

	updateInstalledVersionFn := func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
		status.InstalledVersion = doguResource.Spec.Version
		return status
	}
	doguResource, err = dum.ecosystemClient.Dogus(doguResource.Namespace).
		UpdateStatusWithRetry(ctx, doguResource, updateInstalledVersionFn, metav1.UpdateOptions{})
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
	fromDogu, err := dum.localDoguFetcher.FetchInstalled(ctx, doguResource.Name)
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
