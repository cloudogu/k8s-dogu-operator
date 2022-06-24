package controllers

import (
	"context"
	_ "embed"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

//go:embed testdata/postfix-configuration.yaml
var postfixConfigurationBytes []byte
var postfixConfiguration = &v1.ConfigMap{}

//go:embed testdata/postfix-deployment.yaml
var postfixDeploymentBytes []byte
var postfixDeployment = &appsv1.Deployment{}

func init() {
	err := yaml.Unmarshal(postfixConfigurationBytes, postfixConfiguration)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(postfixDeploymentBytes, postfixDeployment)
	if err != nil {
		panic(err)
	}
}

func Test_doguConfigurationHandler_doUpdate(t *testing.T) {
	namespace := "test"
	doguName := "postfix"
	podName := "postfix-random"
	t.Run("success", func(t *testing.T) {
		// given
		ctx := context.TODO()
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: namespace, Labels: doguLabels},
		}
		err = fakeClient.Create(ctx, pod)
		require.NoError(t, err)

		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		err = fakeClient.Create(ctx, postfixDeployment)
		require.NoError(t, err)

		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.NoError(t, err)
		podNamespaceName := types.NamespacedName{Name: podName, Namespace: namespace}
		err = fakeClient.Get(ctx, podNamespaceName, &v1.Pod{})
		require.NotNil(t, err)
		assert.True(t, errors.IsNotFound(err))

		expectedDeployment := &appsv1.Deployment{}
		deploymentNamespaceName := types.NamespacedName{Name: doguName, Namespace: namespace}
		err = fakeClient.Get(ctx, deploymentNamespaceName, expectedDeployment)
		require.NoError(t, err)

		container := expectedDeployment.Spec.Template.Spec.Containers[0]
		requests := container.Resources.Requests
		expectedCPURequestQuantity, err := resource.ParseQuantity("1")
		require.NoError(t, err)
		assert.True(t, requests.Cpu().Equal(expectedCPURequestQuantity))
		expectedMemoryRequestQuantity, err := resource.ParseQuantity("1Mi")
		require.NoError(t, err)
		assert.True(t, requests.Memory().Equal(expectedMemoryRequestQuantity))

		limits := container.Resources.Limits
		expectedCPULimitQuantity, err := resource.ParseQuantity("1")
		require.NoError(t, err)
		assert.True(t, limits.Cpu().Equal(expectedCPULimitQuantity))
		expectedMemoryLimitQuantity, err := resource.ParseQuantity("1Mi")
		require.NoError(t, err)
		assert.True(t, limits.Memory().Equal(expectedMemoryLimitQuantity))
	})

	t.Run("no installed dogu should not return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		// when
		err := doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.NoError(t, err)
	})

	t.Run("error when getting the configmap should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu configuration configmap")
	})

	t.Run("error when getting the deployment should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		postfixConfiguration.ResourceVersion = ""
		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu deployment")
	})

	t.Run("invalid memory request should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		postfixConfiguration.ResourceVersion = ""
		postfixConfiguration.Data["memory-request"] = "1invalid"
		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		postfixDeployment.ResourceVersion = ""
		err = fakeClient.Create(ctx, postfixDeployment)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse memory request quantity")
	})

	t.Run("invalid cpu request should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		postfixConfiguration.ResourceVersion = ""
		postfixConfiguration.Data["memory-request"] = "1Mi"
		postfixConfiguration.Data["cpu-request"] = "1Mghdgi"
		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		postfixDeployment.ResourceVersion = ""
		err = fakeClient.Create(ctx, postfixDeployment)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse cpu request quantity")
	})

	t.Run("invalid memory limit should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		postfixConfiguration.ResourceVersion = ""
		postfixConfiguration.Data["memory-request"] = "1Mi"
		postfixConfiguration.Data["cpu-request"] = "1"
		postfixConfiguration.Data["memory-limit"] = "2fvdsfMi"
		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		postfixDeployment.ResourceVersion = ""
		err = fakeClient.Create(ctx, postfixDeployment)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse memory limit quantity")
	})

	t.Run("invalid cpu limit should return an error", func(t *testing.T) {
		// given
		ctx := context.TODO()
		cmNamespaceName := types.NamespacedName{Name: "postfix-configuration", Namespace: namespace}
		fakeClient := fake.NewClientBuilder().WithScheme(getDoguConfigScheme()).Build()
		doguConfigurationHandler := &doguConfigurationHandler{Client: fakeClient}

		doguLabels := make(map[string]string)
		doguLabels["dogu"] = "postfix"
		dogu := &k8sv1.Dogu{
			ObjectMeta: metav1.ObjectMeta{Name: doguName, Namespace: namespace, Labels: doguLabels},
		}
		err := fakeClient.Create(ctx, dogu)
		require.NoError(t, err)

		postfixConfiguration.ResourceVersion = ""
		postfixConfiguration.Data["memory-request"] = "1Mi"
		postfixConfiguration.Data["cpu-request"] = "1"
		postfixConfiguration.Data["memory-limit"] = "2Mi"
		postfixConfiguration.Data["cpu-limit"] = "1Mghdgi"
		err = fakeClient.Create(ctx, postfixConfiguration)
		require.NoError(t, err)

		postfixDeployment.ResourceVersion = ""
		err = fakeClient.Create(ctx, postfixDeployment)
		require.NoError(t, err)

		// when
		err = doguConfigurationHandler.doUpdate(ctx, cmNamespaceName)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse cpu limit quantity")
	})
}

func getDoguConfigScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &k8sv1.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &appsv1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &v1.ConfigMap{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}, &v1.Pod{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PodList",
	}, &v1.PodList{})

	return scheme
}
