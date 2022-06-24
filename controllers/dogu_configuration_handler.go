package controllers

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	containerresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
)

const (
	cpuRequestKey    = "cpu_request"
	cpuLimitKey      = "cpu_limit"
	memoryRequestKey = "memory_request"
	memoryLimitKey   = "memory_limit"
)

// doguConfigurationHandler is used to detect changes in configmaps for dogu configuration.
// It implements the event.EventHandler interface.
type doguConfigurationHandler struct {
	client.Client
}

// NewDoguConfigurationHandler creates a new instance for the dogu configuration handler.
func NewDoguConfigurationHandler(client client.Client) *doguConfigurationHandler {
	return &doguConfigurationHandler{client}
}

// Create implements EventHandler.
func (dch *doguConfigurationHandler) Create(evt event.CreateEvent, _ workqueue.RateLimitingInterface) {
	logger := log.FromContext(context.Background())
	logger.Info("Create method")

	if evt.Object != nil {
		name := evt.Object.GetName()

		if !dch.isPotentialDoguConfigMap(name) {
			return
		}

		err := dch.doUpdate(context.Background(), types.NamespacedName{
			Name:      evt.Object.GetName(),
			Namespace: evt.Object.GetNamespace(),
		})

		if err != nil {
			logger.Error(err, "failed to create dogu config")
		}
	}
}

// Update implements EventHandler.
func (dch *doguConfigurationHandler) Update(evt event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	logger := log.FromContext(context.Background())
	logger.Info("Update method")

	if evt.ObjectNew != nil {
		name := evt.ObjectNew.GetName()

		if !dch.isPotentialDoguConfigMap(name) {
			return
		}

		err := dch.doUpdate(context.Background(), types.NamespacedName{
			Name:      name,
			Namespace: evt.ObjectNew.GetNamespace(),
		})

		if err != nil {
			logger.Error(err, "failed to update dogu config")
		}
	}
}

// Delete implements EventHandler.
func (dch *doguConfigurationHandler) Delete(_ event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	// do nothing
	logger := log.FromContext(context.Background())
	logger.Info("Delete method")
}

// Generic implements EventHandler
func (dch *doguConfigurationHandler) Generic(_ event.GenericEvent, _ workqueue.RateLimitingInterface) {
	// do nothing
	logger := log.FromContext(context.Background())
	logger.Info("Generic method")
}

func (dch *doguConfigurationHandler) doUpdate(ctx context.Context, config types.NamespacedName) error {
	logger := log.FromContext(ctx)
	dogu, err := dch.getDoguForConfigurationConfigMap(ctx, config)
	if err != nil {
		logger.Error(err, "failed to get dogu")
		return nil
	}

	cm := &v1.ConfigMap{}
	err = dch.Client.Get(ctx, config, cm)
	if err != nil {
		return fmt.Errorf("failed to get dogu configuration configmap: %w", err)
	}

	deployment := &appsv1.Deployment{}
	namespaceName := types.NamespacedName{Namespace: dogu.Namespace, Name: dogu.Name}
	err = dch.Client.Get(ctx, namespaceName, deployment)
	if err != nil {
		return fmt.Errorf("failed to get dogu deployment: %w", err)
	}

	err = dch.updateDeployment(deployment, cm)
	if err != nil {
		return err
	}

	err = dch.Client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// TODO use dynamic namespace
	err = dch.Client.DeleteAllOf(ctx, &v1.Pod{}, client.InNamespace("ecosystem"), client.MatchingLabels{"dogu": dogu.Name})
	if err != nil {
		return fmt.Errorf("failed to restart pods: %w", err)
	}

	return nil
}

func (dch *doguConfigurationHandler) updateDeployment(deployment *appsv1.Deployment, configmap *v1.ConfigMap) error {
	resourceRequests := make(map[v1.ResourceName]containerresource.Quantity)
	resourceLimits := make(map[v1.ResourceName]containerresource.Quantity)

	strMemRequest, ok := configmap.Data[memoryRequestKey]
	if ok {
		memRequest, err := containerresource.ParseQuantity(strMemRequest)
		if err != nil {
			return fmt.Errorf("failed to parse memory request quantity: %w", err)
		}
		resourceRequests[v1.ResourceMemory] = memRequest
	}

	strCPURequest, ok := configmap.Data[cpuRequestKey]
	if ok {
		cpuRequest, err := containerresource.ParseQuantity(strCPURequest)
		if err != nil {
			return fmt.Errorf("failed to parse cpu request quantity: %w", err)
		}
		resourceRequests[v1.ResourceCPU] = cpuRequest
	}

	strMemLimit, ok := configmap.Data[memoryLimitKey]
	if ok {
		memLimit, err := containerresource.ParseQuantity(strMemLimit)
		if err != nil {
			return fmt.Errorf("failed to parse memory limit quantity: %w", err)
		}
		resourceLimits[v1.ResourceMemory] = memLimit
	}

	strCPULimit, ok := configmap.Data[cpuLimitKey]
	if ok {
		cpuLimit, err := containerresource.ParseQuantity(strCPULimit)
		if err != nil {
			return fmt.Errorf("failed to parse cpu limit quantity: %w", err)
		}
		resourceLimits[v1.ResourceCPU] = cpuLimit
	}

	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = resourceRequests
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = resourceLimits

	return nil
}

func (dch *doguConfigurationHandler) getDoguForConfigurationConfigMap(ctx context.Context, configmap types.NamespacedName) (*k8sv1.Dogu, error) {
	doguName := strings.Split(configmap.Name, "-")[0]
	dogu := &k8sv1.Dogu{}
	objectKey := types.NamespacedName{
		Name:      doguName,
		Namespace: configmap.Namespace,
	}
	err := dch.Client.Get(ctx, objectKey, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu for configmap: %w", err)
	}

	return dogu, nil
}

func (dch *doguConfigurationHandler) belongsDoguToConfigmap(dogu k8sv1.Dogu, configmap string) bool {
	return fmt.Sprintf("%s-configuration", dogu.Name) == configmap
}

func (dch *doguConfigurationHandler) isPotentialDoguConfigMap(configmap string) bool {
	return strings.HasSuffix(configmap, "-configuration")
}
