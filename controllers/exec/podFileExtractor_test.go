package exec

import (
	"bytes"
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLsShellCommand = NewShellCommand("/bin/sh", "-c", "/bin/ls /k8s/ || true")
var testContext = context.TODO()

func Test_podFileExtractor_ExtractK8sResourcesFromContainer(t *testing.T) {
	ldapCr := readLdapDoguResource(t)
	// simulate dogu in a non-default namespace
	ldapCr.Namespace = testNamespace
	ldapDescriptor := readLdapDogu(t)

	t.Run("should fail with command error on exec pod", func(t *testing.T) {
		execPod := NewMockExecPodFactory(t)
		execPod.EXPECT().Exec(testCtx, ldapCr, ldapDescriptor, testLsShellCommand).Return(bytes.NewBufferString("uh oh"), assert.AnError)

		sut := &podFileExtractor{
			factory: execPod,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromExecPod(testContext, ldapCr, ldapDescriptor)

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
	})
	t.Run("should run successfully with file output", func(t *testing.T) {
		execPod := NewMockExecPodFactory(t)
		execPod.EXPECT().Exec(testCtx, ldapCr, ldapDescriptor, testLsShellCommand).Once().Return(bytes.NewBufferString("test-k8s-resources.yaml"), nil)
		expectedCatCommand := &shellCommand{command: "/bin/cat", args: []string{"/k8s/test-k8s-resources.yaml"}}
		execPod.EXPECT().Exec(testCtx, ldapCr, ldapDescriptor, expectedCatCommand).Once().Return(bytes.NewBufferString("resource { content : goes-here }"), nil)

		sut := &podFileExtractor{
			factory: execPod,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromExecPod(testContext, ldapCr, ldapDescriptor)

		// then
		require.NoError(t, err)
		expectedFileMap := make(map[string]string)
		expectedFileMap["/k8s/test-k8s-resources.yaml"] = "resource { content : goes-here }"
		assert.Equal(t, expectedFileMap, actual)
	})
	t.Run("should run successfully without file output", func(t *testing.T) {
		execPod := NewMockExecPodFactory(t)
		execPod.EXPECT().Exec(testCtx, ldapCr, ldapDescriptor, testLsShellCommand).Return(bytes.NewBufferString("No such file or directory"), nil)

		sut := &podFileExtractor{
			factory: execPod,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromExecPod(testContext, ldapCr, ldapDescriptor)

		// then
		require.NoError(t, err)
		assert.Empty(t, actual)
		assert.NotNil(t, actual)
	})
}
