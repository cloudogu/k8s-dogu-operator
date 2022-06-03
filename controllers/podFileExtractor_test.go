package controllers

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const testNamespace = "test-namespace"

var testContext = context.TODO()

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte

func Test_podFileExtractor_createExecPodSpec(t *testing.T) {
	t.Run("should create exec container name with pseudo-unique suffix", func(t *testing.T) {
		ldapCr := readDoguResource(t)
		ldapDogu := readDogu(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getInstallScheme(ldapCr)).
			Build()
		sut := &podFileExtractor{k8sClient: fakeClient}

		// when
		_, containerName, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		assert.Regexp(t, "^ldap-execpod-\\w{6}$", containerName)
	})
	t.Run("should create exec pod same name as container name", func(t *testing.T) {
		ldapCr := readDoguResource(t)
		ldapDogu := readDogu(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getInstallScheme(ldapCr)).
			Build()
		sut := &podFileExtractor{k8sClient: fakeClient}

		// when
		podspec, containerName, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		require.Len(t, podspec.Spec.Containers, 1)
		assert.Equal(t, podspec.Spec.Containers[0].Name, containerName)
	})
	t.Run("should create exec pod from dogu image", func(t *testing.T) {
		ldapCr := readDoguResource(t)
		ldapDogu := readDogu(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getInstallScheme(ldapCr)).
			Build()
		sut := &podFileExtractor{k8sClient: fakeClient}

		// when
		podspec, _, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		require.Len(t, podspec.Spec.Containers, 1)
		assert.Equal(t, podspec.Spec.Containers[0].Image, ldapDogu.Image+":"+ldapDogu.Version)
	})

}

func Test_podFileExtractor_findPod(t *testing.T) {
	t.Run("should find running pod immediately", func(t *testing.T) {
		ldapCr := readDoguResource(t)

		const containerPodName = "letest-execpod-1q2w3e"
		podObjectKey := client.ObjectKey{
			Name:      containerPodName,
			Namespace: testNamespace,
		}
		podSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  containerPodName,
						Image: "le/test:image",
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(getInstallScheme(ldapCr)).
			WithObjects(podSpec).
			Build()
		sut := &podFileExtractor{k8sClient: fakeClient}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		err := sut.findPod(testContext, podObjectKey, containerPodName)

		// then
		require.NoError(t, err)
	})
	t.Run("should return expressive error for unready pod after timeout", func(t *testing.T) {
		ldapCr := readDoguResource(t)

		const containerPodName = "letest-execpod-1q2w3e"
		podObjectKey := client.ObjectKey{
			Name:      containerPodName,
			Namespace: testNamespace,
		}
		podSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerPodName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  containerPodName,
						Image: "le/test:image",
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodFailed},
		}
		fakeClient := fake.NewClientBuilder().
			WithScheme(getInstallScheme(ldapCr)).
			WithObjects(podSpec).
			Build()
		sut := &podFileExtractor{k8sClient: fakeClient}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		err := sut.findPod(testContext, podObjectKey, containerPodName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not come up in time")
		assert.Contains(t, err.Error(), containerPodName)
		assert.Contains(t, err.Error(), "status Failed")
	})
}

func Test_newPodFileExtractor(t *testing.T) {
	t.Run("should implement fileExtractor interface", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().Build()

		// when
		actual := newPodFileExtractor(fakeClient, &rest.Config{}, fake2.NewSimpleClientset())

		// then
		assert.Implements(t, (*fileExtractor)(nil), actual)
	})
}

func readDoguResource(t *testing.T) *k8sv1.Dogu {
	t.Helper()
	ldapCr := &k8sv1.Dogu{}

	err := yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		t.Fatal(err.Error())
	}

	return ldapCr
}

func readDogu(t *testing.T) *core.Dogu {
	t.Helper()

	ldapDogu := &core.Dogu{}
	err := json.Unmarshal(ldapBytes, ldapDogu)
	if err != nil {
		t.Fatal(err.Error())
	}

	return ldapDogu
}

func getInstallScheme(dogu *k8sv1.Dogu) *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, dogu)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &v1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}, &corev1.Pod{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &corev1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &corev1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &corev1.PersistentVolumeClaim{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})

	return scheme
}
