package util

import (
	"bytes"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util/mocks"
	"github.com/stretchr/testify/mock"
	"io"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func Test_exexPod_createPod(t *testing.T) {
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

func Test_commandExecutor_execCmd(t *testing.T) {
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
		_, _, err := commandExecutor.execCmd(command)

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
		actual, actualErr, err := commandExecutor.execCmd(&command)

		// then
		require.NoError(t, err)
		assert.Equal(t, "hallo", actual.String())
		assert.Equal(t, "", actualErr.String())
	})
}

func Test_ExecPod_Exec(t *testing.T) {
	t.Run("should fail with arbitrary error", func(t *testing.T) {
	})
	t.Run("should be successful", func(t *testing.T) {
	})
}

type mockRestExecutor struct {
	mock.Mock
}

func (m *mockRestExecutor) Execute(string, *url.URL, *rest.Config, io.Reader, io.Writer, io.Writer, bool, remotecommand.TerminalSizeQueue) error {
	args := m.Called()
	return args.Error(0)
}
