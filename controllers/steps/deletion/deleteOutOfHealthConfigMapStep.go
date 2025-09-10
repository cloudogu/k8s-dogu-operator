package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const configMapName = "k8s-dogu-operator-dogu-health"

type DeleteOutOfHealthConfigMapStep struct {
	client k8sClient
}

func NewDeleteOutOfHealthConfigMapStep(client k8sClient) *DeleteOutOfHealthConfigMapStep {
	return &DeleteOutOfHealthConfigMapStep{client: client}
}

func (dhc *DeleteOutOfHealthConfigMapStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := dhc.deleteDoguOutOfHealthConfigMap(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}

func (dhc *DeleteOutOfHealthConfigMapStep) deleteDoguOutOfHealthConfigMap(ctx context.Context, dogu *v2.Dogu) error {
	namespace := dogu.Namespace
	stateConfigMap := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Namespace: namespace, Name: configMapName}
	err := dhc.client.Get(ctx, cmKey, stateConfigMap, &client.GetOptions{})

	newData := stateConfigMap.Data
	if err != nil || newData == nil {
		newData = make(map[string]string)
	}
	delete(newData, dogu.Name)

	stateConfigMap.Data = newData

	// Update the ConfigMap
	err = dhc.client.Update(ctx, stateConfigMap, &client.UpdateOptions{})
	return err
}
