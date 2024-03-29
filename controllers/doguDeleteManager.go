package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cesregistry "github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

const finalizerName = "dogu-finalizer"

// doguDeleteManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguDeleteManager struct {
	client                client.Client
	localDoguFetcher      cloudogu.LocalDoguFetcher
	doguRegistrator       cloudogu.DoguRegistrator
	serviceAccountRemover cloudogu.ServiceAccountRemover
	exposedPortRemover    cloudogu.ExposePortRemover
	eventRecorder         record.EventRecorder
}

// NewDoguDeleteManager creates a new instance of doguDeleteManager.
func NewDoguDeleteManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, mgrSet *util.ManagerSet, recorder record.EventRecorder, clientSet kubernetes.Interface) *doguDeleteManager {
	return &doguDeleteManager{
		client:                client,
		localDoguFetcher:      mgrSet.LocalDoguFetcher,
		doguRegistrator:       mgrSet.DoguRegistrator,
		serviceAccountRemover: serviceaccount.NewRemover(cesRegistry, mgrSet.LocalDoguFetcher, mgrSet.CommandExecutor, client, clientSet, operatorConfig.Namespace),
		exposedPortRemover:    resource.NewDoguExposedPortHandler(client),
		eventRecorder:         recorder,
	}
}

// Delete deletes the given dogu along with all those Kubernetes resources that the dogu operator initially created.
func (m *doguDeleteManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	err := doguResource.ChangeStateWithRetry(ctx, m.client, k8sv1.DoguStatusDeleting)
	if err != nil {
		return err
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.localDoguFetcher.FetchInstalled(doguResource.Name)
	if err != nil {
		logger.Error(err, "failed to fetch installed dogu ")
	}

	if dogu != nil {
		logger.Info("Delete service accounts...")
		err = m.serviceAccountRemover.RemoveAll(ctx, dogu)
		if err != nil {
			logger.Error(err, "failed to remove service accounts")
		}

		logger.Info("Unregister dogu...")
		err = m.doguRegistrator.UnregisterDogu(doguResource.Name)
		if err != nil {
			logger.Error(err, "failed to unregister dogu")
		}

		logger.Info("Remove potential exposed ports from loadbalancer...")
		err = m.exposedPortRemover.RemoveExposedPorts(ctx, doguResource, dogu)
		if err != nil {
			logger.Error(err, "failed to remove exposed ports")
		}
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
