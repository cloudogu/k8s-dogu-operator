package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteOutOfHealthConfigMapStep struct {
	client client.Client
}

func NewDeleteOutOfHealthConfigMapStep(client client.Client) *DeleteOutOfHealthConfigMapStep {
	return &DeleteOutOfHealthConfigMapStep{client: client}
}

func (dhc *DeleteOutOfHealthConfigMapStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(dhc.DeleteDoguOutOfHealthConfigMap(ctx, doguResource))
}

func (dhc *DeleteOutOfHealthConfigMapStep) DeleteDoguOutOfHealthConfigMap(ctx context.Context, dogu *v2.Dogu) error {
	namespace := dogu.Namespace
	stateConfigMap := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: namespace, Name: "k8s-dogu-operator-dogu-health"}
	err := dhc.client.Get(ctx, cmKey, stateConfigMap, &client.GetOptions{})

	newData := stateConfigMap.Data
	if err != nil || newData == nil {
		newData = make(map[string]string)
	}
	delete(newData, dogu.Name)

	stateConfigMap.Data = newData

	// Update the ConfigMap
	// _, err = m.k8sClientSet.CoreV1().ConfigMaps(namespace).Update(ctx, stateConfigMap, metav1api.UpdateOptions{})
	err = dhc.client.Update(ctx, stateConfigMap, &client.UpdateOptions{})
	return err
}
