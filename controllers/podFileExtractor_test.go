package controllers

import (
	"context"
	_ "embed"
	"testing"

	utilmocks "github.com/cloudogu/k8s-dogu-operator/controllers/util/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace                  = "test-namespace"
	testLdapPodContainerNamePrefix = "ldap"
	testPodContainerNameSuffix     = "1q2w3e"
	testLdapPodContainerName       = testLdapPodContainerNamePrefix + "-execpod-" + testPodContainerNameSuffix
)

var testLdapExecPodKey = newObjectKey(testNamespace, testLdapPodContainerName)

var testContext = context.TODO()

func Test_podFileExtractor_ExtractK8sResourcesFromContainer(t *testing.T) {
	ldapCr := readDoguCr(t, ldapCrBytes)
	// simulate dogu in a non-default namespace
	ldapCr.Namespace = testNamespace
	ldapExecPodKey := &client.ObjectKey{Namespace: testNamespace, Name: testLdapPodContainerName}

	t.Run("should fail with command error on exec pod", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("", assert.AnError)
		execPod := utilmocks.NewExecPod(t)
		execPod.On("ObjectKey").Return(ldapExecPodKey)

		sut := &podFileExtractor{
			k8sClient: fakeClient,
			clientSet: clientset,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, execPod)

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should run successfully with file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("test-k8s-resources.yaml", nil)
		expectedCatCommand := []string{"/bin/cat", "/k8s/test-k8s-resources.yaml"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedCatCommand).Return("resource { content : goes-here }", nil)
		execPod := utilmocks.NewExecPod(t)
		execPod.On("ObjectKey").Return(ldapExecPodKey)

		sut := &podFileExtractor{
			k8sClient: fakeClient,
			clientSet: clientset,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, execPod)

		// then
		require.NoError(t, err)
		expectedFileMap := make(map[string]string)
		expectedFileMap["/k8s/test-k8s-resources.yaml"] = "resource { content : goes-here }"
		assert.Equal(t, expectedFileMap, actual)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should run successfully without file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("No such file or directory", nil)
		execPod := utilmocks.NewExecPod(t)
		execPod.On("ObjectKey").Return(ldapExecPodKey)

		sut := &podFileExtractor{
			k8sClient: fakeClient,
			clientSet: clientset,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, execPod)

		// then
		require.NoError(t, err)
		assert.Empty(t, actual)
		assert.NotNil(t, actual)
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

func newObjectKey(namespace, name string) *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
}
