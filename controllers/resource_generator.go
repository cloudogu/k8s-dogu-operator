package controllers

import (
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceGenerator struct {
}

func (r *ResourceGenerator) GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) *appsv1.Deployment {
	// Create deployment
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      doguResource.Name,
		Namespace: doguResource.Namespace,
	}}

	labels := map[string]string{"dogu": doguResource.Name}
	deployment.ObjectMeta.Labels = labels
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Strategy: appsv1.DeploymentStrategy{
			Type: "Recreate",
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Hostname: doguResource.Name,
				Containers: []corev1.Container{{
					Name:            doguResource.Name,
					Image:           dogu.Image + ":" + dogu.Version,
					ImagePullPolicy: corev1.PullIfNotPresent}},
			},
		},
	}

	return deployment
}

func (r *ResourceGenerator) GetDoguService(doguResource *k8sv1.Dogu) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguResource.Name,
			Namespace: doguResource.Namespace,
			Labels:    map[string]string{"app": cesLabel, "dogu": doguResource.Name},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"dogu": doguResource.Name},
		},
	}

	return service
}
