package controllers

import (
	"context"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	reg "github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

// doguInstallManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguInstallManager struct {
	client                client.Client
	recorder              record.EventRecorder
	doguRemoteRegistry    cesremote.Registry
	doguLocalRegistry     cesregistry.DoguRegistry
	localDoguFetcher      internal.LocalDoguFetcher
	resourceDoguFetcher   internal.ResourceDoguFetcher
	imageRegistry         internal.ImageRegistry
	doguRegistrator       internal.DoguRegistrator
	dependencyValidator   internal.DependencyValidator
	serviceAccountCreator internal.ServiceAccountCreator
	doguSecretHandler     internal.DoguSecretHandler
	fileExtractor         internal.FileExtractor
	collectApplier        internal.CollectApplier
	resourceUpserter      internal.ResourceUpserter
	execPodFactory        internal.ExecPodFactory
}

// NewDoguInstallManager creates a new instance of doguInstallManager.
func NewDoguInstallManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, eventRecorder record.EventRecorder) (*doguInstallManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}

	localDoguFetcher := reg.NewLocalDoguFetcher(cesRegistry.DoguRegistry())
	resourceDoguFetcher := reg.NewResourceDoguFetcher(client, doguRemoteRegistry)
	imageRegistry := imageregistry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)
	limitPatcher := limit.NewDoguDeploymentLimitPatcher(cesRegistry)
	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))
	upserter := resource.NewUpserter(client, limitPatcher)

	fileExtract := exec.NewPodFileExtractor(client, restConfig, clientSet)
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}

	doguRegistrator := reg.NewCESDoguRegistrator(client, cesRegistry, resourceGenerator)
	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())

	executor := exec.NewCommandExecutor(client, clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor, client)
	collectApplier := resource.NewCollectApplier(applier)

	return &doguInstallManager{
		client:                client,
		recorder:              eventRecorder,
		localDoguFetcher:      localDoguFetcher,
		resourceDoguFetcher:   resourceDoguFetcher,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		doguSecretHandler:     resource.NewDoguSecretsWriter(client, cesRegistry),
		fileExtractor:         fileExtract,
		collectApplier:        collectApplier,
		resourceUpserter:      upserter,
		execPodFactory:        exec.NewExecPodFactory(client, restConfig, executor),
	}, nil
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m *doguInstallManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	doguResource.Status = k8sv1.DoguStatus{RequeueTime: doguResource.Status.RequeueTime, Status: k8sv1.DoguStatusInstalling, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.client)
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

	logger.Info("Register dogu...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
	err = m.doguRegistrator.RegisterNewDogu(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	logger.Info("Write dogu secrets from setup...")
	err = m.doguSecretHandler.WriteDoguSecretsToRegistry(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to write dogu secrets from setup: %w", err)
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

	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
	anExecPod, err := m.execPodFactory.NewExecPod(internal.VolumeModeInstall, doguResource, dogu)
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
	customDeployment, err := m.applyCustomK8sResources(ctx, customK8sResources, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Create dogu resources...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to create dogu resources: %w", err)
	}

	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = doguResource.Update(ctx, m.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, m.client)
		if err != nil {
			return fmt.Errorf("failed to delete development dogu map from cluster: %w", err)
		}
	}

	return nil
}

func (m *doguInstallManager) applyCustomK8sResources(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	return m.collectApplier.CollectApply(ctx, customK8sResources, doguResource)
}

func (m *doguInstallManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile, patchingDeployment *appsv1.Deployment) error {
	err := m.resourceUpserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, patchingDeployment)
	if err != nil {
		return fmt.Errorf("failed to create resource(s) for dogu %s: %w", dogu.Name, err)
	}

	return nil
}

func deleteExecPod(ctx context.Context, execPod internal.ExecPod, recorder record.EventRecorder, doguResource *k8sv1.Dogu) {
	err := execPod.Delete(ctx)
	if err != nil {
		recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Failed to delete execPod %s: %w", execPod.PodName(), err)
	}
}
