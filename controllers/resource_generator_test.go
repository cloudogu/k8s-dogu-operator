package controllers

import (
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})

	generator := NewResourceGenerator(scheme)
	t.Run("Return simple deployment", func(t *testing.T) {
		expectedDeployment := getExpectedDeployment()

		actualDeployment, err := generator.GetDoguDeployment(doguCr, ldapDogu)

		require.NoError(t, err)
		assert.Equal(t, expectedDeployment, actualDeployment)
	})
}

func TestResourceGenerator_GetDoguService(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})

	generator := NewResourceGenerator(scheme)
	t.Run("Return simple service", func(t *testing.T) {
		actualService, err := generator.GetDoguService(doguCr, imageConf)
		assert.NoError(t, err)

		referenceFlag := true
		expectedService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ldap",
				Namespace: "clusterns",
				Labels:    map[string]string{"app": cesLabel, "dogu": "ldap"},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         "k8s.cloudogu.com/v1",
					Kind:               "Dogu",
					Name:               "ldap",
					UID:                "",
					Controller:         &referenceFlag,
					BlockOwnerDeletion: &referenceFlag,
				}},
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

func TestResourceGenerator_GetDoguPVC(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})

	generator := NewResourceGenerator(scheme)
	t.Run("Return simple pvc", func(t *testing.T) {
		expectedPVC := getExpectedPVC()
		actualPVC, err := generator.GetDoguPVC(doguCr)
		require.NoError(t, err)
		assert.Equal(t, expectedPVC, actualPVC)
	})
}

func TestResourceGenerator_GetDoguSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &k8sv1.Dogu{})

	generator := NewResourceGenerator(scheme)
	t.Run("Return secret", func(t *testing.T) {
		expectedSecret := getExpectedSecret()
		actualSecret, err := generator.GetDoguSecret(doguCr, map[string]string{"key": "value"})
		require.NoError(t, err)
		assert.Equal(t, expectedSecret, actualSecret)
	})
}

func getExpectedSecret() *corev1.Secret {
	referenceFlag := true
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name:      "ldap-private",
		Namespace: "clusterns",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "k8s.cloudogu.com/v1",
			Kind:               "Dogu",
			Name:               "ldap",
			UID:                "",
			Controller:         &referenceFlag,
			BlockOwnerDeletion: &referenceFlag,
		}},
		Labels: map[string]string{"app": cesLabel, "dogu": "ldap"}},
		StringData: map[string]string{"key": "value"}}
}

func getExpectedPVC() *corev1.PersistentVolumeClaim {
	referenceFlag := true
	doguPvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "clusterns",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "k8s.cloudogu.com/v1",
				Kind:               "Dogu",
				Name:               "ldap",
				UID:                "",
				Controller:         &referenceFlag,
				BlockOwnerDeletion: &referenceFlag,
			}},
		},
	}

	doguPvc.ObjectMeta.Labels = map[string]string{"app": cesLabel, "dogu": "ldap"}
	doguPvc.Spec = corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse("5Gi"),
			},
		},
	}

	return doguPvc
}

func getExpectedDeployment() *appsv1.Deployment {
	labels := map[string]string{"dogu": "ldap"}
	fsGroup := int64(101)
	fsGroupPolicy := corev1.FSGroupChangeOnRootMismatch
	secretPermission := int32(0744)
	referenceFlag := true
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ldap",
			Namespace: "clusterns",
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "k8s.cloudogu.com/v1",
				Kind:               "Dogu",
				Name:               "ldap",
				UID:                "",
				Controller:         &referenceFlag,
				BlockOwnerDeletion: &referenceFlag,
			}},
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
					SecurityContext:  &corev1.PodSecurityContext{FSGroup: &fsGroup, FSGroupChangePolicy: &fsGroupPolicy},
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
								SecretName:  "ldap-private",
								DefaultMode: &secretPermission,
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
