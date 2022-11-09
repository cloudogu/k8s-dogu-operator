package exec

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testLsShellCommand = NewShellCommand("/bin/sh", "-c", "/bin/ls /k8s/ || true")
var testContext = context.TODO()

func Test_podFileExtractor_ExtractK8sResourcesFromContainer(t *testing.T) {
	ldapCr := readLdapDoguResource(t)
	// simulate dogu in a non-default namespace
	ldapCr.Namespace = testNamespace

	t.Run("should fail with command error on exec pod", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()

		execPod := NewExecPodMock(t)
		execPod.On("Exec", testCtx, testLsShellCommand).Return("uh oh", assert.AnError)

		sut := &podFileExtractor{
			k8sClient: fakeClient,
			clientSet: clientset,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, execPod)

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
	})
	t.Run("should run successfully with file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		execPod := NewExecPodMock(t)
		execPod.On("Exec", testCtx, testLsShellCommand).Once().Return("test-k8s-resources.yaml", nil)
		expectedCatCommand := &ShellCommand{Command: "/bin/cat", Args: []string{"/k8s/test-k8s-resources.yaml"}}
		execPod.On("Exec", testCtx, expectedCatCommand).Once().Return("resource { content : goes-here }", nil)

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
	})
	t.Run("should run successfully without file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		execPod := NewExecPodMock(t)
		execPod.On("Exec", testCtx, testLsShellCommand).Return("No such file or directory", nil)

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