package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
	registryConfig "github.com/cloudogu/k8s-registry-lib/config"
	registryErrors "github.com/cloudogu/k8s-registry-lib/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

const finalizerName = "dogu-finalizer"

// doguDeleteManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguDeleteManager struct {
	client                  client.Client
	localDoguFetcher        cloudogu.LocalDoguFetcher
	doguRegistrator         cloudogu.DoguRegistrator
	serviceAccountRemover   cloudogu.ServiceAccountRemover
	exposedPortRemover      cloudogu.ExposePortRemover
	eventRecorder           record.EventRecorder
	doguConfigRepository    thirdParty.DoguConfigRepository
	sensitiveDoguRepository thirdParty.DoguConfigRepository
}

// NewDoguDeleteManager creates a new instance of doguDeleteManager.
func NewDoguDeleteManager(
	client client.Client,
	operatorConfig *config.OperatorConfig,
	mgrSet *util.ManagerSet,
	recorder record.EventRecorder,
	configRepos util.ConfigRepositories,
) *doguDeleteManager {
	return &doguDeleteManager{
		client:                  client,
		localDoguFetcher:        mgrSet.LocalDoguFetcher,
		doguRegistrator:         mgrSet.DoguRegistrator,
		serviceAccountRemover:   serviceaccount.NewRemover(configRepos.SensitiveDoguRepository, mgrSet.LocalDoguFetcher, mgrSet.CommandExecutor, client, mgrSet.ClientSet, operatorConfig.Namespace),
		exposedPortRemover:      resource.NewDoguExposedPortHandler(client),
		eventRecorder:           recorder,
		doguConfigRepository:    configRepos.DoguConfigRepository,
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
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
	dogu, err := m.localDoguFetcher.FetchInstalled(ctx, doguResource.Name)
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
		err = m.doguRegistrator.UnregisterDogu(ctx, doguResource.Name)
		if err != nil {
			logger.Error(err, "failed to unregister dogu")
		}

		logger.Info("Remove potential exposed ports from loadbalancer...")
		err = m.exposedPortRemover.RemoveExposedPorts(ctx, doguResource, dogu)
		if err != nil {
			logger.Error(err, "failed to remove exposed ports")
		}

		logger.Info("Remove health state out of ConfigMap")
		err = m.DeleteDoguOutOfHealthConfigMap(ctx, doguResource)
		if err != nil {
			logger.Error(err, "failed to remove health state out of configMap")
		}

		logger.Info("Remove dogu config and sensitive dogu config...")
		err = m.removeConfigs(ctx, doguResource.Name)
		if err != nil {
			logger.Error(err, "failed to remove configs for dogu", "dogu", doguResource.Name)
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

func (m *doguDeleteManager) DeleteDoguOutOfHealthConfigMap(ctx context.Context, dogu *k8sv1.Dogu) error {
	namespace := dogu.Namespace
	stateConfigMap := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: namespace, Name: "k8s-dogu-operator-dogu-health"}
	err := m.client.Get(ctx, cmKey, stateConfigMap, &client.GetOptions{})

	newData := stateConfigMap.Data
	if err != nil || newData == nil {
		newData = make(map[string]string)
	}
	delete(newData, dogu.Name)

	stateConfigMap.Data = newData

	// Update the ConfigMap
	//_, err = m.k8sClientSet.CoreV1().ConfigMaps(namespace).Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
	err = m.client.Update(ctx, stateConfigMap, &client.UpdateOptions{})
	return err
}

func (m *doguDeleteManager) removeConfigs(ctx context.Context, doguName string) error {
	simpleDoguName := registryConfig.SimpleDoguName(doguName)

	var err error

	if lErr := m.doguConfigRepository.Delete(ctx, simpleDoguName); lErr != nil && !registryErrors.IsNotFoundError(lErr) {
		err = errors.Join(err, fmt.Errorf("could not delete dogu config: %w", lErr))
	}

	if lErr := m.sensitiveDoguRepository.Delete(ctx, simpleDoguName); lErr != nil && !registryErrors.IsNotFoundError(lErr) {
		err = errors.Join(err, fmt.Errorf("could not delete sensitive dogu config: %w", lErr))
	}

	return err
}
