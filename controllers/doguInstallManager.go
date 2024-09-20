package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/errors"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
)

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

// doguInstallManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguInstallManager struct {
	client                  thirdParty.K8sClient
	ecosystemClient         ecoSystem.EcoSystemV1Alpha1Interface
	recorder                record.EventRecorder
	localDoguFetcher        cloudogu.LocalDoguFetcher
	resourceDoguFetcher     cloudogu.ResourceDoguFetcher
	imageRegistry           cloudogu.ImageRegistry
	doguRegistrator         cloudogu.DoguRegistrator
	dependencyValidator     cloudogu.DependencyValidator
	serviceAccountCreator   cloudogu.ServiceAccountCreator
	fileExtractor           cloudogu.FileExtractor
	collectApplier          cloudogu.CollectApplier
	resourceUpserter        cloudogu.ResourceUpserter
	execPodFactory          cloudogu.ExecPodFactory
	doguConfigRepository    thirdParty.DoguConfigRepository
	sensitiveDoguRepository thirdParty.DoguConfigRepository
}

// NewDoguInstallManager creates a new instance of doguInstallManager.
func NewDoguInstallManager(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder, configRepos util.ConfigRepositories) *doguInstallManager {
	return &doguInstallManager{
		client:                  client,
		ecosystemClient:         mgrSet.EcosystemClient,
		recorder:                eventRecorder,
		localDoguFetcher:        mgrSet.LocalDoguFetcher,
		resourceDoguFetcher:     mgrSet.ResourceDoguFetcher,
		imageRegistry:           mgrSet.ImageRegistry,
		doguRegistrator:         mgrSet.DoguRegistrator,
		dependencyValidator:     mgrSet.DependencyValidator,
		serviceAccountCreator:   mgrSet.ServiceAccountCreator,
		fileExtractor:           mgrSet.FileExtractor,
		collectApplier:          mgrSet.CollectApplier,
		resourceUpserter:        mgrSet.ResourceUpserter,
		execPodFactory:          exec.NewExecPodFactory(client, mgrSet.RestConfig, mgrSet.CommandExecutor),
		doguConfigRepository:    configRepos.DoguConfigRepository,
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
	}
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m *doguInstallManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) (err error) {
	logger := log.FromContext(ctx)

	err = doguResource.ChangeStateWithRetry(ctx, m.client, k8sv1.DoguStatusInstalling)
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

	err = doguResource.ChangeStateWithRetry(ctx, m.client, k8sv1.DoguStatusInstalled)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	updateInstalledVersionFn := func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
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

	return nil
}

func (m *doguInstallManager) applyCustomK8sResources(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) error {
	return m.collectApplier.CollectApply(ctx, customK8sResources, doguResource)
}

func (m *doguInstallManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile) error {
	_, err := m.resourceUpserter.UpsertDoguService(ctx, doguResource, imageConfig)
	if err != nil {
		return err
	}

	_, err = m.resourceUpserter.UpsertDoguExposedService(ctx, doguResource, dogu)
	if err != nil {
		return err
	}

	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
	anExecPod, err := m.execPodFactory.NewExecPod(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create ExecPod resource %s: %w", anExecPod.ObjectKey().Name, err)
	}
	err = anExecPod.Create(ctx)
	if err != nil {
		return fmt.Errorf("failed to create ExecPod %s: %w", anExecPod.ObjectKey().Name, err)
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
			lErr := m.doguConfigRepository.Delete(lCtx, config.SimpleDoguName(doguName))
			if lErr != nil && !errors.IsNotFoundError(lErr) {
				logger.Error(lErr, "could not delete dogu config during cleanUp", "dogu", doguName)
			} else {
				logger.Info("deleted dogu config during cleanUp", "dogu", doguName)
			}
		}

		if !sensitiveCfgAlreadyExists {
			lErr := m.sensitiveDoguRepository.Delete(lCtx, config.SimpleDoguName(doguName))
			if lErr != nil && !errors.IsNotFoundError(lErr) {
				logger.Error(lErr, "could not delete sensitive dogu config during cleanUp", "dogu", doguName)
			} else {
				logger.Info("deleted sensitive dogu config during cleanUp", "dogu", doguName)
			}
		}
	}

	emptyCfg := config.CreateDoguConfig(config.SimpleDoguName(doguName), make(config.Entries))

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

func deleteExecPod(ctx context.Context, execPod cloudogu.ExecPod, recorder record.EventRecorder, doguResource *k8sv1.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
