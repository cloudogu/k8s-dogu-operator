package resource

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-registry-lib/config"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	triggerSyncKey = "sync_resource_requirements"
)

// RequirementsUpdater is responsible to update all resource requirements for dogu deployments when a certain trigger is called.
type RequirementsUpdater struct {
	client              client.Client
	namespace           string
	globalConfigWatcher globalConfigurationWatcher
	doguFetcher         cloudogu.LocalDoguFetcher
	requirementsGen     requirementsGenerator
}

// NewRequirementsUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewRequirementsUpdater(client client.Client, namespace string, doguConfigGetter doguConfigGetter, doguFetcher cloudogu.LocalDoguFetcher, globalWatcher globalConfigurationWatcher) (*RequirementsUpdater, error) {
	requirementsGen := NewRequirementsGenerator(doguConfigGetter)

	return &RequirementsUpdater{
		client:              client,
		namespace:           namespace,
		globalConfigWatcher: globalWatcher,
		doguFetcher:         doguFetcher,
		requirementsGen:     requirementsGen,
	}, nil
}

// Start is the entry point for the updater.
func (hlu *RequirementsUpdater) Start(ctx context.Context) error {
	return hlu.startWatch(ctx)
}

func (hlu *RequirementsUpdater) startWatch(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Start watching on the trigger for synchronization [%s] in global config", triggerSyncKey))

	watchResChan, err := hlu.globalConfigWatcher.Watch(ctx, config.KeyFilter(triggerSyncKey))
	if err != nil {
		return fmt.Errorf("could not start watch for key [%s]: %w", triggerSyncKey, err)
	}

	for range watchResChan {
		lErr := hlu.triggerSync(ctx)
		if lErr != nil {
			return lErr
		}
	}

	ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Stopped watching on global config for certificate key [%s] because channel has been closed", triggerSyncKey))

	return nil
}

func (hlu *RequirementsUpdater) triggerSync(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info("Trigger for updating dogu resource requirements detected in global config. Updating deployment for all dogus...")

	installedDogus, err := hlu.getInstalledDogus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installed dogus from the cluster: %w", err)
	}

	var result error
	for _, dogu := range installedDogus.Items {
		doguJson, lErr := hlu.doguFetcher.FetchInstalled(ctx, dogu.GetName())
		if lErr != nil {
			result = errors.Join(result, fmt.Errorf("failed to get dogu.json of dogu [%s] from registry: %w", dogu.Name, lErr))
			continue
		}

		doguIdentifier := types.NamespacedName{Name: dogu.GetName(), Namespace: dogu.GetNamespace()}
		doguDeployment := &v1.Deployment{}
		lErr = hlu.client.Get(ctx, doguIdentifier, doguDeployment)
		if lErr != nil {
			result = errors.Join(result, fmt.Errorf("failed to get deployment of dogu [%s/%s] from cluster: %w", dogu.Namespace, dogu.Name, lErr))
			continue
		}

		requirements, lErr := hlu.requirementsGen.Generate(ctx, doguJson)
		if lErr != nil {
			result = errors.Join(result, fmt.Errorf("failed to generate resource requirements of dogu [%s/%s] in cluster: %w", dogu.Namespace, dogu.Name, lErr))
			continue
		}

		doguDeployment.Spec.Template.Spec.Containers[0].Resources = requirements

		lErr = hlu.client.Update(ctx, doguDeployment)
		if lErr != nil {
			result = errors.Join(result, fmt.Errorf("failed to update deployment of dogu [%s/%s] in cluster: %w", dogu.Namespace, dogu.Name, lErr))
			continue
		}
	}

	return result
}

func (hlu *RequirementsUpdater) getInstalledDogus(ctx context.Context) (*k8sv1.DoguList, error) {
	doguList := &k8sv1.DoguList{}

	err := hlu.client.List(ctx, doguList, client.InNamespace(hlu.namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list dogus in namespace [%s]: %w", hlu.namespace, err)
	}

	return doguList, nil
}
