package health

import (
	"context"
	"fmt"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

const statusUpdateEventReason = "HealthStatusUpdate"

type DoguStatusUpdater struct {
	ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface
	recorder        record.EventRecorder
	k8sClientSet    thirdParty.ClientSet
}

func NewDoguStatusUpdater(ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface, recorder record.EventRecorder, k8sClientSet thirdParty.ClientSet) *DoguStatusUpdater {
	return &DoguStatusUpdater{
		ecosystemClient: ecosystemClient,
		recorder:        recorder,
		k8sClientSet:    k8sClientSet,
	}
}

// UpdateStatus sets the health status of the dogu according to whether if it's available or not.
func (dsw *DoguStatusUpdater) UpdateStatus(ctx context.Context, doguName types.NamespacedName, isAvailable bool) error {
	doguClient := dsw.ecosystemClient.Dogus(doguName.Namespace)

	dogu, err := doguClient.Get(ctx, doguName.Name, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu resource %q: %w", doguName, err)
	}

	desiredHealthStatus := doguv1.UnavailableHealthStatus
	if isAvailable {
		desiredHealthStatus = doguv1.AvailableHealthStatus
	}

	_, err = doguClient.UpdateStatusWithRetry(ctx, dogu, func(status doguv1.DoguStatus) doguv1.DoguStatus {
		status.Health = desiredHealthStatus
		return status
	}, metav1api.UpdateOptions{})

	if err != nil {
		message := fmt.Sprintf("failed to update dogu %q with current health status [%q] to desired health status [%q]", doguName, dogu.Status.Health, desiredHealthStatus)
		dsw.recorder.Event(dogu, v1.EventTypeWarning, statusUpdateEventReason, message)
		return fmt.Errorf("%s: %w", message, err)
	}

	dsw.recorder.Eventf(dogu, v1.EventTypeNormal, statusUpdateEventReason, "successfully updated health status to %q", desiredHealthStatus)
	return nil
}

func (dsw *DoguStatusUpdater) UpdateHealthConfigMap(ctx context.Context, doguDeployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error {
	namespace := doguDeployment.Namespace

	// Read out ConfigMap
	stateConfigMap, err := dsw.k8sClientSet.CoreV1().ConfigMaps(namespace).Get(ctx, "k8s-dogu-operator-dogu-health", metav1api.GetOptions{})
	//TODO error handling

	// Get all pods to deployment
	pods, err := dsw.k8sClientSet.CoreV1().Pods(namespace).List(ctx, metav1api.ListOptions{
		LabelSelector: metav1api.FormatLabelSelector(doguDeployment.Spec.Selector),
	})
	//TODO error handling

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

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, doguDeployment.Name) && isState {
			newData := stateConfigMap.Data
			if err != nil || newData == nil {
				newData = make(map[string]string)
			}
			for _, status := range pod.Status.ContainerStatuses {
				newData[doguDeployment.Name] = ""
				if *status.Started {
					newData[doguDeployment.Name] = state
					break
				}
			}
			stateConfigMap.Data = newData

			// Update the ConfigMap
			_, err = dsw.k8sClientSet.CoreV1().ConfigMaps(namespace).Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
			if err != nil {
				log.FromContext(ctx).Error(err, "failed to remove health state out of configMap")
			}
			break
		}
	}

	return nil
}
