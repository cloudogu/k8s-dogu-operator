package manager

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

const restartedAtAnnotationKey = "k8s.cloudogu.com/restartedAt"

type doguRestartManager struct {
	doguInterface       doguInterface
	deploymentInterface deploymentInterface
}

func NewDoguRestartManager(doguClient doguClient.DoguInterface, deploymentClient appsv1.DeploymentInterface) DoguRestartManager {
	return &doguRestartManager{
		doguInterface:       doguClient,
		deploymentInterface: deploymentClient,
	}
}

func (drm *doguRestartManager) RestartDogu(ctx context.Context, dogu *v2.Dogu) error {
	deployment, err := drm.deploymentInterface.Get(ctx, dogu.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}

	deployment.Spec.Template.Annotations[restartedAtAnnotationKey] = time.Now().Format(time.RFC3339)

	_, err = drm.deploymentInterface.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
