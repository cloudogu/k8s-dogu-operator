package controllers

import (
	"context"
	"fmt"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
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

// doguInstallManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguDeleteManager struct {
	client.Client
	Scheme                *runtime.Scheme
	DoguRemoteRegistry    cesremote.Registry
	DoguLocalRegistry     cesregistry.DoguRegistry
	ImageRegistry         imageRegistry
	DoguRegistrator       doguRegistrator
	ServiceAccountRemover serviceAccountRemover
	DoguSecretHandler     doguSecretHandler
}

// NewDoguDeleteManager creates a new instance of doguDeleteManager.
func NewDoguDeleteManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry) (*doguDeleteManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed find create new remote dogu registry: %w", err)
	}

	resourceGenerator := resource.NewResourceGenerator(client.Scheme())

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed find cluster config: %w", err)
	}
	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())

	return &doguDeleteManager{
		Client:                client,
		Scheme:                client.Scheme(),
		DoguRemoteRegistry:    doguRemoteRegistry,
		DoguLocalRegistry:     cesRegistry.DoguRegistry(),
		DoguRegistrator:       NewCESDoguRegistrator(client, cesRegistry, resourceGenerator),
		ServiceAccountRemover: serviceaccount.NewRemover(cesRegistry, executor),
	}, nil
}

func (m *doguDeleteManager) getDoguDescriptorFromLocalRegistry(doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	dogu, err := m.DoguLocalRegistry.Get(doguResource.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from local dogu registry: %w", err)
	}

	return dogu, nil
}

func (m *doguDeleteManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.Client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.getDoguDescriptorFromLocalRegistry(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	logger.Info("Delete service accounts...")
	err = m.ServiceAccountRemover.RemoveAll(ctx, doguResource.Namespace, dogu)
	if err != nil {
		logger.Error(err, "failed to remove service accounts")
	}

	logger.Info("Unregister dogu...")
	err = m.DoguRegistrator.UnregisterDogu(doguResource.Name)
	if err != nil {
		logger.Error(err, "failed to unregister dogu")
	}

	logger.Info("Remove finalizer...")
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err = m.Client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been : %s", doguResource.Namespace, doguResource.Name, controllerutil.OperationResultUpdated))

	return nil
}
