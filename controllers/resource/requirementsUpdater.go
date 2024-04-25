package resource

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/controllers/localregistry"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	coreosclient "go.etcd.io/etcd/client/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	triggerSyncEtcdKeyFullPath = "/config/_global/sync_resource_requirements"
)

// requirementsUpdater is responsible to update all resource requirements for dogu deployments when a certain trigger is called.
type requirementsUpdater struct {
	client            client.Client
	namespace         string
	registry          registry.Registry
	localDoguRegistry localregistry.LocalDoguRegistry
	requirementsGen   cloudogu.ResourceRequirementsGenerator
}

// NewRequirementsUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewRequirementsUpdater(client client.Client, namespace string, ecosystemClientSet ecoSystem.EcoSystemV1Alpha1Interface, clientSet kubernetes.Interface) (*requirementsUpdater, error) {
	endpoint := fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", namespace)
	reg, err := registry.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{endpoint},
	})
	if err != nil {
		return nil, err
	}

	requirementsGen := NewRequirementsGenerator(reg)

	return &requirementsUpdater{
		client:    client,
		namespace: namespace,
		registry:  reg,
		localDoguRegistry: localregistry.NewCombinedLocalDoguRegistry(
			ecosystemClientSet.Dogus(namespace),
			clientSet.CoreV1().ConfigMaps(namespace),
			reg,
		),
		requirementsGen: requirementsGen,
	}, nil
}

// Start is the entry point for the updater.
func (hlu *requirementsUpdater) Start(ctx context.Context) error {
	return hlu.startEtcdWatch(ctx)
}

func (hlu *requirementsUpdater) startEtcdWatch(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Start etcd watcher on certificate key [%s]", triggerSyncEtcdKeyFullPath))

	triggerChannel := make(chan *coreosclient.Response)
	go func() {
		hlu.registry.RootConfig().Watch(ctx, triggerSyncEtcdKeyFullPath, false, triggerChannel)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-triggerChannel:
			err := hlu.triggerSync(ctx)
			if err != nil {
				return err
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

		requirements, err := hlu.requirementsGen.Generate(doguJson)
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
