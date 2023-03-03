package limit

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	"github.com/hashicorp/go-multierror"
	coreosclient "go.etcd.io/etcd/client/v2"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	triggerSyncEtcdKeyFullPath = "/config/_global/trigger_container_limit_sync"
)

// hardwareLimitUpdater is responsible to update all hardware limits for dogu deployments when a certain trigger is called.
type hardwareLimitUpdater struct {
	client           client.Client
	namespace        string
	registry         registry.Registry
	doguLimitPatcher cloudogu.LimitPatcher
}

// doguLimits contains all data necessary to limit the physical resources for a dogu.
type doguLimits struct {
	// cpuLimit Sets the cpu requests and limit values for the dogu deployment to the contained value. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	cpuLimit *resource.Quantity
	// memoryLimit Sets the memory requests and limit values for the dogu deployment to the contained value. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	memoryLimit *resource.Quantity
	// ephemeralStorageLimit Sets the ephemeral storage requests and limit values for the dogu deployment to the contained value. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	ephemeralStorageLimit *resource.Quantity
}

func (d *doguLimits) CpuLimit() *resource.Quantity {
	return d.cpuLimit
}

func (d *doguLimits) MemoryLimit() *resource.Quantity {
	return d.memoryLimit
}

func (d *doguLimits) EphemeralStorageLimit() *resource.Quantity {
	return d.ephemeralStorageLimit
}

// NewHardwareLimitUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewHardwareLimitUpdater(client client.Client, namespace string) (*hardwareLimitUpdater, error) {
	endpoint := fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", namespace)
	reg, err := registry.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{endpoint},
	})
	if err != nil {
		return nil, err
	}

	return &hardwareLimitUpdater{
		client:           client,
		namespace:        namespace,
		registry:         reg,
		doguLimitPatcher: NewDoguDeploymentLimitPatcher(reg),
	}, nil
}

// Start is the entry point for the updater.
func (hlu *hardwareLimitUpdater) Start(ctx context.Context) error {
	return hlu.startEtcdWatch(ctx)
}

func (hlu *hardwareLimitUpdater) startEtcdWatch(ctx context.Context) error {
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

func (hlu *hardwareLimitUpdater) triggerSync(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info("Trigger for updating dogu hardware limits detected in registry. Updating deployment for all dogus...")

	installedDogus, err := hlu.getInstalledDogus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installed dogus from the cluster: %w", err)
	}

	var result error
	for _, dogu := range installedDogus.Items {
		doguIdentifier := types.NamespacedName{Name: dogu.GetName(), Namespace: dogu.GetNamespace()}
		doguDeployment := &v1.Deployment{}
		err = hlu.client.Get(ctx, doguIdentifier, doguDeployment)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to get deployment of dogu [%s/%s]: %w", doguIdentifier.Namespace, doguIdentifier.Name, err))
			continue
		}

		limits, err := hlu.doguLimitPatcher.RetrievePodLimits(&dogu)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to retrieve memory limits of dogu [%s/%s]: %w", doguIdentifier.Namespace, doguIdentifier.Name, err))
			continue
		}

		ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Updating deployment for dogu [%s] with limits [%+v]", dogu.GetName(), limits))
		err = hlu.doguLimitPatcher.PatchDeployment(doguDeployment, limits)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to change deployment of dogu [%s/%s]: %w", doguIdentifier.Namespace, doguIdentifier.Name, err))
			continue
		}

		err = hlu.client.Update(ctx, doguDeployment)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("failed to update deployment of dogu [%s/%s] in cluster: %w", doguIdentifier.Namespace, doguIdentifier.Name, err))
			continue
		}
	}

	return result
}

func (hlu *hardwareLimitUpdater) getInstalledDogus(ctx context.Context) (*k8sv1.DoguList, error) {
	doguList := &k8sv1.DoguList{}

	err := hlu.client.List(ctx, doguList, client.InNamespace(hlu.namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list dogus in namespace [%s]: %w", hlu.namespace, err)
	}

	return doguList, nil
}
