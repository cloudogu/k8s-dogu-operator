package resource

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	triggerSyncKey = "sync_resource_requirements"
)

// requirementsUpdater is responsible to update all resource requirements for dogu deployments when a certain trigger is called.
type requirementsUpdater struct {
	client              client.Client
	namespace           string
	globalConfigWatcher GlobalConfigurationWatcher
	localDoguRegistry   DoguGetter
	requirementsGen     RequirementsGenerator
}

// NewRequirementsUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewRequirementsUpdater(client client.Client, namespace string, provider DoguConfigProvider, doguReg DoguGetter, globalWatcher GlobalConfigurationWatcher) (*requirementsUpdater, error) {
	requirementsGen := NewRequirementsGenerator(provider)

	return &requirementsUpdater{
		client:              client,
		namespace:           namespace,
		globalConfigWatcher: globalWatcher,
		localDoguRegistry:   doguReg,
		requirementsGen:     requirementsGen,
	}, nil
}

// Start is the entry point for the updater.
func (hlu *requirementsUpdater) Start(ctx context.Context) error {
	return hlu.startWatch(ctx)
}

func (hlu *requirementsUpdater) startWatch(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Start watching on global config for certificate key [%s]", triggerSyncKey))

	watch, err := hlu.globalConfigWatcher.Watch(ctx, triggerSyncKey, false)
	if err != nil {
		return fmt.Errorf("could not start watch for key [%s]", triggerSyncKey)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, open := <-watch.ResultChan:
			if !open {
				ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Stopped watching on global config for certificate key [%s] because channel is closed", triggerSyncKey))
				return nil
			}

			lErr := hlu.triggerSync(ctx)
			if lErr != nil {
				return lErr
			}
		}
	}
}

func (hlu *requirementsUpdater) triggerSync(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info("Trigger for updating dogu resource requirements detected in registry. Updating deployment for all dogus...")

	installedDogus, err := hlu.getInstalledDogus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installed dogus from the cluster: %w", err)
	}

	var result error
	for _, dogu := range installedDogus.Items {
		doguJson, err := hlu.localDoguRegistry.GetCurrent(ctx, dogu.GetName())
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to get dogu.json of dogu [%s] from registry: %w", dogu.Name, err))
			continue
		}

		doguIdentifier := types.NamespacedName{Name: dogu.GetName(), Namespace: dogu.GetNamespace()}
		doguDeployment := &v1.Deployment{}
		err = hlu.client.Get(ctx, doguIdentifier, doguDeployment)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to get deployment of dogu [%s/%s] from cluster: %w", dogu.Namespace, dogu.Name, err))
			continue
		}

		requirements, err := hlu.requirementsGen.Generate(ctx, doguJson)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to generate resource requirements of dogu [%s/%s] in cluster: %w", dogu.Namespace, dogu.Name, err))
			continue
		}

		doguDeployment.Spec.Template.Spec.Containers[0].Resources = requirements

		err = hlu.client.Update(ctx, doguDeployment)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("failed to update deployment of dogu [%s/%s] in cluster: %w", dogu.Namespace, dogu.Name, err))
			continue
		}
	}

	return result
}

func (hlu *requirementsUpdater) getInstalledDogus(ctx context.Context) (*k8sv1.DoguList, error) {
	doguList := &k8sv1.DoguList{}

	err := hlu.client.List(ctx, doguList, client.InNamespace(hlu.namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list dogus in namespace [%s]: %w", hlu.namespace, err)
	}

	return doguList, nil
}
