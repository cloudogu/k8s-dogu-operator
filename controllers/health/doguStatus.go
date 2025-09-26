package health

import (
	"context"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
)

const statusUpdateEventReason = "HealthStatusUpdate"
const healthConfigMapName = "k8s-dogu-operator-dogu-health"

type DoguStatusUpdater struct {
	ecosystemClient doguClient.EcoSystemV2Interface
	recorder        record.EventRecorder
	k8sClientSet    clientSet
}

func NewDoguStatusUpdater(ecosystemClient doguClient.EcoSystemV2Interface, recorder record.EventRecorder, k8sClientSet kubernetes.Interface) *DoguStatusUpdater {
	return &DoguStatusUpdater{
		ecosystemClient: ecosystemClient,
		recorder:        recorder,
		k8sClientSet:    k8sClientSet,
	}
}

func (dsw *DoguStatusUpdater) UpdateHealthConfigMap(ctx context.Context, doguDeployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error {
	namespace := doguDeployment.Namespace

	stateConfigMap, err := dsw.k8sClientSet.CoreV1().ConfigMaps(namespace).Get(ctx, healthConfigMapName, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get health state configMap: %w", err)
	}
	initHealthConfigMap(stateConfigMap, doguDeployment)

	// Get all pods to deployment
	pods, err := dsw.k8sClientSet.CoreV1().Pods(namespace).List(ctx, metav1api.ListOptions{
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

		_, err = dsw.k8sClientSet.CoreV1().ConfigMaps(namespace).Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update health state in health configMap: %w", err)
		}

		if stateConfigMap.Data[doguDeployment.Name] != "" {
			break
		}
	}

	return nil
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
