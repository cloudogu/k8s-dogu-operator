package controllers

import (
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"testing"
)

import _ "embed"

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte
var ldapDogu = &core.Dogu{}

//go:embed testdata/image-config.json
var imageConfBytes []byte
var imageConf = &imagev1.ConfigFile{}

//go:embed testdata/dogu_cr.json
var doguCrBytes []byte
var doguCr = &k8sv1.Dogu{}

func init() {
	err := json.Unmarshal(ldapBytes, ldapDogu)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(imageConfBytes, imageConf)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(doguCrBytes, doguCr)
	if err != nil {
		panic(err)
	}
}

func TestResourceGenerator_GetDoguDeployment(t *testing.T) {
	generator := ResourceGenerator{}
	t.Run("Return simple deployment", func(t *testing.T) {
		expectedDeployment := getExpectedDeployment()
		actualDeployment := generator.GetDoguDeployment(doguCr, ldapDogu)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {
	generator := ResourceGenerator{}
	t.Run("Return simple service", func(t *testing.T) {
		actualService, err := generator.GetDoguService(doguCr, imageConf)

		assert.NoError(t, err)

		expectedService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: "clusterns",
				Labels:    map[string]string{"app": cesLabel, "dogu": "ldap"},
			},
			Spec: corev1.ServiceSpec{
				Type:     corev1.ServiceTypeClusterIP,
				Selector: map[string]string{"dogu": "ldap"},
				Ports: []corev1.ServicePort{
					{Name: "80", Port: 80, Protocol: "TCP"},
				},
			},
		}
		assert.Equal(t, expectedService, actualService)
	})
}

func getExpectedDeployment() *appsv1.Deployment {
	labels := map[string]string{"dogu": "ldap"}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "clusterns",
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{
				Type: "Recreate",
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry-cloudogu-com"}},
					Hostname:         "ldap",
					Volumes: []corev1.Volume{{
						Name: "node-master-file",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: "node-master-file"},
							},
						},
					}, {
						Name: "ldap-private",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "ldap-private",
							},
						},
					}, {
						Name: "ldap-data",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "ldap",
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "ldap",
						Image:           "registry.cloudogu.com/official/ldap:2.4.48-4",
						ImagePullPolicy: corev1.PullIfNotPresent,
						VolumeMounts: []corev1.VolumeMount{
							{Name: "node-master-file", ReadOnly: true, MountPath: "/etc/ces/node_master", SubPath: "node_master"},
							{Name: "ldap-private", ReadOnly: true, MountPath: "/private", SubPath: ""},
							{Name: "ldap-data", ReadOnly: false, MountPath: "/var/lib/openldap", SubPath: "db"},
							{Name: "ldap-data", ReadOnly: false, MountPath: "/etc/openldap/slapd.d", SubPath: "config"}},
					}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}
}
