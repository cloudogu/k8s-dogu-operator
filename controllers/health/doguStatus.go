package health

import (
	"context"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

const healthConfigMapName = "k8s-dogu-operator-dogu-health"

type DoguStatusUpdater struct {
	recorder           record.EventRecorder
	configMapInterface configMapInterface
	podInterface       podInterface
}

func NewDoguStatusUpdater(recorder record.EventRecorder, configMapInterface corev1.ConfigMapInterface, podInterface corev1.PodInterface) *DoguStatusUpdater {
	return &DoguStatusUpdater{
		recorder:           recorder,
		configMapInterface: configMapInterface,
		podInterface:       podInterface,
	}
}

func (dsw *DoguStatusUpdater) UpdateHealthConfigMap(ctx context.Context, doguDeployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error {
	stateConfigMap, err := dsw.configMapInterface.Get(ctx, healthConfigMapName, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get health state configMap: %w", err)
	}
	initHealthConfigMap(stateConfigMap, doguDeployment)

	// Get all pods to deployment
	pods, err := dsw.podInterface.List(ctx, metav1api.ListOptions{
		LabelSelector: metav1api.FormatLabelSelector(doguDeployment.Spec.Selector),
	})
	if err != nil {
		return fmt.Errorf("failed to get all pods for the deployment %v: %w", doguDeployment, err)
	}

	isState, state := hasHealthCheckofTypeState(doguJson)

	for _, pod := range pods.Items {
		if isState {
			setHealthConfigMapStateWhenStarted(stateConfigMap, pod, doguDeployment, state)
		}

		_, err = dsw.configMapInterface.Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update health state in health configMap: %w", err)
		}

		if stateConfigMap.Data[doguDeployment.Name] != "" {
			break
		}
	}

	return nil
}

func (dsw *DoguStatusUpdater) DeleteDoguOutOfHealthConfigMap(ctx context.Context, dogu *v2.Dogu) error {
	stateConfigMap, err := dsw.getOrCreateHealthConfigMap(ctx)
	if err != nil {
		return err
	}

	newData := stateConfigMap.Data
	if newData == nil {
		newData = make(map[string]string)
	}
	delete(newData, dogu.Name)

	stateConfigMap.Data = newData

	// Update the ConfigMap
	_, err = dsw.configMapInterface.Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
	return err
}

func (dsw *DoguStatusUpdater) getOrCreateHealthConfigMap(ctx context.Context) (*v1.ConfigMap, error) {
	cm, err := dsw.configMapInterface.Get(ctx, healthConfigMapName, metav1api.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		cm = &v1.ConfigMap{
			ObjectMeta: metav1api.ObjectMeta{Name: healthConfigMapName},
			Data:       map[string]string{},
		}
		cm, err = dsw.configMapInterface.Create(ctx, cm, metav1api.CreateOptions{})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return cm, nil
}

func initHealthConfigMap(stateConfigMap *v1.ConfigMap, doguDeployment *appsv1.Deployment) {
	if stateConfigMap.Data == nil {
		stateConfigMap.Data = make(map[string]string)
	}
	stateConfigMap.Data[doguDeployment.Name] = ""
}

func hasHealthCheckofTypeState(doguJson *cesappcore.Dogu) (bool, string) {
	isState := false
	state := "ready"
	for _, healthCheck := range doguJson.HealthChecks {
		if healthCheck.Type == "state" {
			isState = true
			if healthCheck.State != "" {
				state = healthCheck.State
			}
			break
		}
	}
	return isState, state
}

func setHealthConfigMapStateWhenStarted(stateConfigMap *v1.ConfigMap, pod v1.Pod, doguDeployment *appsv1.Deployment, state string) {
	for _, status := range pod.Status.ContainerStatuses {
		if *status.Started {
			stateConfigMap.Data[doguDeployment.Name] = state
			break
		}
	}
}
