package controllers

import (
	"context"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	cesreg "github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "dogu-finalizer"

// doguDeleteManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguDeleteManager struct {
	client                client.Client
	scheme                *runtime.Scheme
	doguRemoteRegistry    cesremote.Registry
	doguLocalRegistry     cesregistry.DoguRegistry
	imageRegistry         imageRegistry
	doguRegistrator       doguRegistrator
	serviceAccountRemover serviceAccountRemover
	doguSecretHandler     doguSecretHandler
}

// NewDoguDeleteManager creates a new instance of doguDeleteManager.
func NewDoguDeleteManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry) (*doguDeleteManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}
	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())

	return &doguDeleteManager{
		client:                client,
		scheme:                client.Scheme(),
		doguRemoteRegistry:    doguRemoteRegistry,
		doguLocalRegistry:     cesRegistry.DoguRegistry(),
		doguRegistrator:       cesreg.NewCESDoguRegistrator(client, cesRegistry, resourceGenerator),
		serviceAccountRemover: serviceaccount.NewRemover(cesRegistry, executor),
	}, nil
}

func (m *doguDeleteManager) getDoguDescriptorFromLocalRegistry(doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	dogu, err := m.doguLocalRegistry.Get(doguResource.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from local dogu registry: %w", err)
	}

	return dogu, nil
}

// Delete deletes the given dogu along with all those Kubernetes resources that the dogu operator initially created.
func (m *doguDeleteManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.getDoguDescriptorFromLocalRegistry(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	logger.Info("Delete service accounts...")
	err = m.serviceAccountRemover.RemoveAll(ctx, doguResource.Namespace, dogu)
	if err != nil {
		logger.Error(err, "failed to remove service accounts")
	}

	logger.Info("Unregister dogu...")
	err = m.doguRegistrator.UnregisterDogu(doguResource.Name)
	if err != nil {
		logger.Error(err, "failed to unregister dogu")
	}

	// delete custom descriptor
	doguConfigMap, err := getDoguConfigMap(ctx, m.client, doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu config map: %w", err)
	}

	err = deleteDoguConfigMap(ctx, m.client, doguConfigMap)
	if err != nil {
		logger.Error(err, "failed to delete custom dogu configmap %s: %w", doguResource.Name, err)
	}

	logger.Info("Remove finalizer...")
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err = m.client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been : %s", doguResource.Namespace, doguResource.Name, controllerutil.OperationResultUpdated))

	return nil
}
