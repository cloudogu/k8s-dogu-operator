package controllers

import (
	"context"
	"errors"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const restartedAtAnnotationKey = "k8s.cloudogu.com/restartedAt"

type doguRestartManager struct {
	doguInterface       doguInterface
	client              client.Client
	deploymentInterface deploymentInterface
}

func NewDoguRestartManager(ecosystemClient ecosystemInterface, clientSet ClientSet, client client.Client, namespace string) *doguRestartManager {
	deployInt := clientSet.AppsV1().Deployments(namespace)
	return &doguRestartManager{
		doguInterface:       ecosystemClient.Dogus(namespace),
		client:              client,
		deploymentInterface: deployInt,
	}
}

func (drm *doguRestartManager) RestartAllDogus(ctx context.Context) error {
	doguList, err := drm.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	var errs []error

	for _, dogu := range doguList.Items {
		err := drm.RestartDogu(ctx, &dogu)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (drm *doguRestartManager) RestartDogu(ctx context.Context, dogu *v2.Dogu) error {
	deployment, err := dogu.GetDeployment(ctx, drm.client)
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
