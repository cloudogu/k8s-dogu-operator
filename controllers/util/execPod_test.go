package util

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util/mocks"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

const testNamespace = "ecosystem"
const podName = "le-test-pod-name"
const containerName = "ldap"

func Test_defaultSufficeGenerator_String(t *testing.T) {
	actual := (&defaultSufficeGenerator{}).String(6)
	assert.Len(t, actual, 6)
}

func TestExecPod_ObjectKey(t *testing.T) {
	// given
	inputResource := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "le-dogu", Namespace: testNamespace},
	}
	sut := &execPod{podName: podName, doguResource: inputResource}

	// when
	actual := sut.ObjectKey()

	// then
	assert.NotEmpty(t, actual)
	expected := &client.ObjectKey{
		Namespace: testNamespace,
		Name:      podName,
	}
	assert.Equal(t, expected, actual)
}

func Test_execPod_createPod(t *testing.T) {
	ldapDogu := readLdapDogu(t)
	ldapDoguResource := readLdapDoguResource(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(getTestScheme()).
		Build()
	sut := &execPod{client: fakeClient, doguResource: ldapDoguResource, dogu: ldapDogu}

	t.Run("should create exec pod same name as container name", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Name, containerName)
	})

	t.Run("should create exec pod from dogu image", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Image, ldapDogu.Image+":"+ldapDogu.Version)
	})
}

func Test_execPod_Exec(t *testing.T) {
	t.Run("should fail with arbitrary error", func(t *testing.T) {
		// given
		ldapDogu := readLdapDogu(t)
		ldapDoguResource := readLdapDoguResource(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		cmd := &resource.ShellCommand{Command: "/bin/ls", Args: []string{"-lahF"}}
		mockExec := mocks.NewCommandExecutor(t)
		outBuf := bytes.NewBufferString("")
		errBuf := bytes.NewBufferString("oh noez!")
		mockExec.On("ExecCmd", cmd).Return(outBuf, errBuf, assert.AnError)
		sut := &execPod{client: fakeClient, doguResource: ldapDoguResource, dogu: ldapDogu, executor: mockExec}

		// when
		actualOut, actualErrOut, err := sut.Exec(cmd)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Empty(t, actualOut)
		assert.Equal(t, "oh noez!", actualErrOut)
	})
	t.Run("should be successful", func(t *testing.T) {
		// given
		ldapDogu := readLdapDogu(t)
		ldapDoguResource := readLdapDoguResource(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		cmd := &resource.ShellCommand{Command: "/bin/ls", Args: []string{"-lahF"}}
		mockExec := mocks.NewCommandExecutor(t)
		outBuf := bytes.NewBufferString("possibly some output goes here")
		errBuf := bytes.NewBufferString("")
		mockExec.On("ExecCmd", cmd).Return(outBuf, errBuf, nil)
		sut := &execPod{client: fakeClient, doguResource: ldapDoguResource, dogu: ldapDogu, executor: mockExec}

		// when
		actualOut, actualErrOut, err := sut.Exec(cmd)

		// then
		require.NoError(t, err)
		assert.Equal(t, "possibly some output goes here", actualOut)
		assert.Equal(t, "", actualErrOut)
	})
}

func Test_commandExecutor_ExecCmd(t *testing.T) {
	command := &resource.ShellCommand{
		Command: "/bin/ls",
		Args:    []string{"/home"},
	}
	t.Run("should run command with error on failed container", func(t *testing.T) {
		// given
		runner := mocks.NewRunner(t)
		runner.On("Run").Return(createStreams(), assert.AnError)
		runner.On("SetCommand", command).Return()
		commandExecutor := defaultCommandExecutor{runner: runner}

		// when
		_, _, err := commandExecutor.ExecCmd(command)

		// then
		require.Error(t, err)
	})
	t.Run("should run successfully", func(t *testing.T) {
		// given
		stream := genericclioptions.IOStreams{
			Out:    bytes.NewBufferString("hallo"),
			ErrOut: &bytes.Buffer{},
		}
		runner := mocks.NewRunner(t)
		runner.On("Run").Return(stream, nil)
		runner.On("SetCommand", command).Return()
		commandExecutor := defaultCommandExecutor{runner: runner}
		command := resource.ShellCommand{
			Command: "/bin/ls",
			Args:    []string{"/home"},
		}

		// when
		actual, actualErr, err := commandExecutor.ExecCmd(&command)

		// then
		require.NoError(t, err)
		assert.Equal(t, "hallo", actual.String())
		assert.Equal(t, "", actualErr.String())
	})
}
