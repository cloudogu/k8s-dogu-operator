package controllers

import (
	"context"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-commons-lib/errors"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/go-logr/logr"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

// doguInstallManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguInstallManager struct {
	client                        K8sClient
	ecosystemClient               doguClient.EcoSystemV2Interface
	recorder                      record.EventRecorder
	localDoguFetcher              localDoguFetcher
	resourceDoguFetcher           resourceDoguFetcher
	imageRegistry                 imageRegistry
	doguRegistrator               doguRegistrator
	dependencyValidator           upgrade.DependencyValidator
	serviceAccountCreator         serviceAccountCreator
	fileExtractor                 exec.FileExtractor
	collectApplier                resource.CollectApplier
	resourceUpserter              resource.ResourceUpserter
	execPodFactory                exec.ExecPodFactory
	doguConfigRepository          doguConfigRepository
	sensitiveDoguRepository       doguConfigRepository
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
}

// NewDoguInstallManager creates a new instance of doguInstallManager.
func NewDoguInstallManager(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder, configRepos util.ConfigRepositories) *doguInstallManager {
	return &doguInstallManager{
		client:                        client,
		ecosystemClient:               mgrSet.EcosystemClient,
		recorder:                      eventRecorder,
		localDoguFetcher:              mgrSet.LocalDoguFetcher,
		resourceDoguFetcher:           mgrSet.ResourceDoguFetcher,
		imageRegistry:                 mgrSet.ImageRegistry,
		doguRegistrator:               mgrSet.DoguRegistrator,
		dependencyValidator:           mgrSet.DependencyValidator,
		serviceAccountCreator:         mgrSet.ServiceAccountCreator,
		fileExtractor:                 mgrSet.FileExtractor,
		collectApplier:                mgrSet.CollectApplier,
		resourceUpserter:              mgrSet.ResourceUpserter,
		execPodFactory:                exec.NewExecPodFactory(client, mgrSet.RestConfig, mgrSet.CommandExecutor),
		doguConfigRepository:          configRepos.DoguConfigRepository,
		sensitiveDoguRepository:       configRepos.SensitiveDoguRepository,
		securityValidator:             mgrSet.SecurityValidator,
		doguAdditionalMountsValidator: mgrSet.DoguAdditionalMountValidator,
	}
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m *doguInstallManager) Install(ctx context.Context, doguResource *doguv2.Dogu) (err error) {
	logger := log.FromContext(ctx)

	err = doguResource.ChangeStateWithRetry(ctx, m.client, doguv2.DoguStatusInstalling)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	// Set the finalizer at the beginning of the install procedure.
	// This is required because an error during installation would leave a dogu resource with its
	// k8s resources in the cluster. A delete would tidy up those resources but would not start the
	// delete procedure from the controller.
	logger.Info("Add dogu finalizer...")
	controllerutil.AddFinalizer(doguResource, finalizerName)
	err = m.client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, developmentDoguMap, err := m.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Check dogu dependencies...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
	err = m.dependencyValidator.ValidateDependencies(ctx, dogu)
	if err != nil {
		return err
	}

	logger.Info("Validating dogu security...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
	err = m.securityValidator.ValidateSecurity(dogu, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Validating dogu additional mounts...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
	err = m.doguAdditionalMountsValidator.ValidateAdditionalMounts(ctx, dogu, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Create dogu config and sensitive dogu config...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
	cleanUp, err := m.createConfigs(ctx, doguResource.Name, logger)
	defer func() {
		cleanUp(err)
	}()
	if err != nil {
		return fmt.Errorf("failed to create configs for dogu: %w", err)
	}

	logger.Info("Register dogu...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
	err = m.doguRegistrator.RegisterNewDogu(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	logger.Info("Create service accounts...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
	err = m.serviceAccountCreator.CreateAll(ctx, dogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	logger.Info("Pull image config...")
	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", dogu.Image+":"+dogu.Version)
	imageConfig, err := m.imageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to pull image config: %w", err)
	}

	logger.Info("Create dogu resources...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig)
	if err != nil {
		return fmt.Errorf("failed to create dogu resources: %w", err)
	}

	err = doguResource.ChangeStateWithRetry(ctx, m.client, doguv2.DoguStatusInstalled)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	updateInstalledVersionFn := func(status doguv2.DoguStatus) doguv2.DoguStatus {
		status.InstalledVersion = doguResource.Spec.Version
		return status
	}
	doguResource, err = m.ecosystemClient.Dogus(doguResource.Namespace).
		UpdateStatusWithRetry(ctx, doguResource, updateInstalledVersionFn, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update dogu installed version: %w", err)
	}

	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, m.client)
		if err != nil {
			return fmt.Errorf("failed to delete development dogu map from cluster: %w", err)
		}
	}

	// Update Status for DataVolume
	logger.Info("Set Default Data Volume Size...")
	err = SetDefaultDataVolumeSize(ctx, m.client, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update Dogu-Status: %w", err)
	}

	return nil
}

func (m *doguInstallManager) applyCustomK8sResources(ctx context.Context, customK8sResources map[string]string, doguResource *doguv2.Dogu) error {
	return m.collectApplier.CollectApply(ctx, customK8sResources, doguResource)
}

func (m *doguInstallManager) createDoguResources(ctx context.Context, doguResource *doguv2.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile) error {
	_, err := m.resourceUpserter.UpsertDoguService(ctx, doguResource, dogu, imageConfig)
	if err != nil {
		return err
	}

	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
	anExecPod, err := m.execPodFactory.NewExecPod(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create execPod resource %s: %w", anExecPod.ObjectKey().Name, err)
	}
	err = anExecPod.Create(ctx)
	if err != nil {
		return fmt.Errorf("failed to create execPod %s: %w", anExecPod.ObjectKey().Name, err)
	}
	defer deleteExecPod(ctx, anExecPod, m.recorder, doguResource)

	customK8sResources, err := m.fileExtractor.ExtractK8sResourcesFromContainer(ctx, anExecPod)
	if err != nil {
		return fmt.Errorf("failed to pull customK8sResources: %w", err)
	}

	if len(customK8sResources) > 0 {
		m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", util.GetMapKeysAsString(customK8sResources))
	}
	err = m.applyCustomK8sResources(ctx, customK8sResources, doguResource)
	if err != nil {
		return err
	}

	_, err = m.resourceUpserter.UpsertDoguPVCs(ctx, doguResource, dogu)
	if err != nil {
		return err
	}

	_, err = m.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return err
	}

	err = m.resourceUpserter.UpsertDoguNetworkPolicies(ctx, doguResource, dogu)
	if err != nil {
		return err
	}

	return nil
}

func (m *doguInstallManager) createConfigs(ctx context.Context, doguName string, logger logr.Logger) (func(error), error) {
	var doguCfgAlreadyExists, sensitiveCfgAlreadyExists bool

	cleanUp := func(err error) {
		if err == nil {
			return
		}

		lCtx := context.WithoutCancel(ctx)

		if !doguCfgAlreadyExists {
			lErr := m.doguConfigRepository.Delete(lCtx, cescommons.SimpleName(doguName))
			if lErr != nil && !errors.IsNotFoundError(lErr) {
				logger.Error(lErr, "could not delete dogu config during cleanUp", "dogu", doguName)
			} else {
				logger.Info("deleted dogu config during cleanUp", "dogu", doguName)
			}
		}

		if !sensitiveCfgAlreadyExists {
			lErr := m.sensitiveDoguRepository.Delete(lCtx, cescommons.SimpleName(doguName))
			if lErr != nil && !errors.IsNotFoundError(lErr) {
				logger.Error(lErr, "could not delete sensitive dogu config during cleanUp", "dogu", doguName)
			} else {
				logger.Info("deleted sensitive dogu config during cleanUp", "dogu", doguName)
			}
		}
	}

	emptyCfg := config.CreateDoguConfig(cescommons.SimpleName(doguName), make(config.Entries))

	_, err := m.doguConfigRepository.Create(ctx, emptyCfg)
	if err != nil {
		if !errors.IsAlreadyExistsError(err) {
			return cleanUp, fmt.Errorf("could not create dogu config for dogu %s: %w", doguName, err)
		}

		doguCfgAlreadyExists = true
	}

	_, err = m.sensitiveDoguRepository.Create(ctx, emptyCfg)
	if err != nil {
		if !errors.IsAlreadyExistsError(err) {
			return cleanUp, fmt.Errorf("could not create sensitive dogu config for dogu %s: %w", doguName, err)
		}

		sensitiveCfgAlreadyExists = true
	}

	return cleanUp, nil
}

// SetCurrentDataVolumeSize set the default DataVolumeSize within the status of the dogu
func SetDefaultDataVolumeSize(ctx context.Context, client client.Client, doguResource *doguv2.Dogu) error {
	logger := log.FromContext(ctx)

	// Check min size condition
	condition := metav1.Condition{
		Type:               doguv2.DoguStatusConditionMeetsMinimumDataVolumeSize,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "DefaultVolumeSizeNotMeetsMinDataSize",
	}
	minDataSize, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		logger.Error(err, "failed to get min data volume size")
		return err
	}

	doguResource.Status.DataVolumeSize.Set(minDataSize.Value())

	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)

	// Update resource
	err = client.Status().Update(ctx, doguResource)
	if err != nil {
		logger.Error(err, "failed to update data volume size")
		return err
	}

	return nil
}

func deleteExecPod(ctx context.Context, execPod exec.ExecPod, recorder record.EventRecorder, doguResource *doguv2.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
