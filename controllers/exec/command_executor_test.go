package exec

import (
	"bytes"
	"context"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/mock"
	"net/url"
	"strings"
	"testing"
	"time"

	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const commandOutput = "username:user"

func TestCommandExecutor_ExecCommandForDogu(t *testing.T) {
	ctx := context.TODO()
	doguResource := readLdapDoguResource(t)
	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	unreadyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionFalse}}},
	}
	runningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	notRunningPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	command := NewShellCommand("ls", "-l")
	originalMaxTries := maxTries
	defer func() { maxTries = originalMaxTries }()
	maxTries = 1

	fakeNewSPDYExecutor, fakeErrorInitNewSPDYExecutor, fakeErrorStreamNewSPDYExecutor := createFakeExecutors(t)

	oldConfigFunc := config.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return nil
	}
	defer func() {
		ctrl.GetConfigOrDie = oldConfigFunc
	}()

	t.Run("success with expected status ContainersStarted", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, runningPod).
			Build()
		clientSet := testclient.NewSimpleClientset(runningPod)
		sut := NewCommandExecutor(cli, clientSet, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor
		expectedBuffer := bytes.NewBufferString(commandOutput)

		// when
		buffer, err := sut.ExecCommandForDogu(ctx, doguResource, command, ContainersStarted)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})

	t.Run("success with retry if dogu was unavailable", func(t *testing.T) {
		// given
		doguResource := readLdapDoguResource(t)
		doguResource.Status.Health = k8sv2.UnavailableHealthStatus
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, runningPod).
			Build()
		err := cli.Update(ctx, doguResource)
		require.NoError(t, err)
		clientSet := testclient.NewSimpleClientset(runningPod)
		sut := NewCommandExecutor(cli, clientSet, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor
		expectedBuffer := bytes.NewBufferString(commandOutput)

		timer := time.NewTimer(time.Second * 3)
		go func() {
			<-timer.C
			doguResource.Status.Health = k8sv2.AvailableHealthStatus
			err := cli.Update(ctx, doguResource)
			require.NoError(t, err)
		}()

		// when
		buffer, err := sut.ExecCommandForDogu(ctx, doguResource, command, ContainersStarted)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})

	t.Run("success with expected status PodReady", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, readyPod).
			Build()
		clientSet := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, clientSet, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor
		expectedBuffer := bytes.NewBufferString("username:user")

		// when
		buffer, err := sut.ExecCommandForDogu(ctx, doguResource, command, PodReady)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})

	t.Run("found no dogu resource", func(t *testing.T) {
		// given
		oldWaitLimit := waitLimit
		waitLimit = time.Second * 3
		defer func() { waitLimit = oldWaitLimit }()

		cli := fake2.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		client := testclient.NewSimpleClientset()
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, nil, "")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get pod for dogu ldap")
	})

	t.Run("found no dogu pod", func(t *testing.T) {
		// given
		oldWaitLimit := waitLimit
		waitLimit = time.Second * 3
		defer func() { waitLimit = oldWaitLimit }()

		cli := fake2.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(doguResource).Build()
		client := testclient.NewSimpleClientset()
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, nil, "")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get pod for dogu ldap")
	})

	t.Run("pod is not ready", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, unreadyPod).
			Build()
		client := testclient.NewSimpleClientset(unreadyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, nil, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "an error occurred while waiting for pod ldap-xyz to have status ready")
		assert.ErrorContains(t, err, "the maximum number of retries was reached")
		assert.ErrorContains(t, err, "expected status ready not fulfilled")
	})

	t.Run("pod is not running", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, notRunningPod).
			Build()
		client := testclient.NewSimpleClientset(notRunningPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, nil, ContainersStarted)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "an error occurred while waiting for pod ldap-xyz to have status started")
		assert.ErrorContains(t, err, "the maximum number of retries was reached")
		assert.ErrorContains(t, err, "expected status started not fulfilled")
	})

	t.Run("failed to create spdy", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, readyPod).
			Build()
		client := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeErrorInitNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, command, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create new spdy executor")
	})

	t.Run("failed to exec stream", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, readyPod).
			Build()
		client := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeErrorStreamNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForDogu(ctx, doguResource, command, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, assert.AnError.Error())
	})
}

func TestExposedCommandExecutor_ExecCommandForPod(t *testing.T) {
	ctx := context.TODO()
	doguResource := readLdapDoguResource(t)
	originalMaxTries := maxTries
	defer func() { maxTries = originalMaxTries }()
	maxTries = 1

	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	unreadyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: doguResource.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionFalse}}},
	}

	command := NewShellCommand("ls", "-l")

	fakeNewSPDYExecutor, fakeErrorInitNewSPDYExecutor, fakeErrorStreamNewSPDYExecutor := createFakeExecutors(t)

	oldConfigFunc := config.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return nil
	}
	defer func() {
		ctrl.GetConfigOrDie = oldConfigFunc
	}()

	t.Run("success", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, readyPod).
			Build()
		clientSet := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, clientSet, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor
		expectedBuffer := bytes.NewBufferString("username:user")

		// when
		buffer, err := sut.ExecCommandForPod(ctx, readyPod, command, PodReady)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})
	t.Run("success with stdin", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, readyPod).
			Build()
		clientSet := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, clientSet, &fake.RESTClient{})
		reader := strings.NewReader("abc")
		buffer := bytes.NewBuffer([]byte{})
		bufferErr := bytes.NewBuffer([]byte{})

		sut.commandExecutorCreator = func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
			mockExecutor := newMockRemoteExecutor(t)
			mockExecutor.EXPECT().StreamWithContext(mock.Anything, remotecommand.StreamOptions{
				// expects the reader as stream option in the mocked call to verify the stdin command
				Stdin:  reader,
				Stdout: buffer,
				Stderr: bufferErr,
				Tty:    false,
			}).RunAndReturn(streamWithContextRun())
			return mockExecutor, nil
		}

		expectedBuffer := bytes.NewBufferString("username:user")
		stdinCmd := NewShellCommandWithStdin(reader, "base64")

		// when
		buffer, err := sut.ExecCommandForPod(ctx, readyPod, stdinCmd, PodReady)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})
	t.Run("found no pods", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().Build()
		client := testclient.NewSimpleClientset()
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForPod(ctx, readyPod, &shellCommand{}, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, `pods "ldap-xyz" not found`)
	})
	t.Run("pod is not ready", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(doguResource, unreadyPod).
			Build()
		client := testclient.NewSimpleClientset(unreadyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForPod(ctx, unreadyPod, nil, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "an error occurred while waiting for pod ldap-xyz to have status ready")
	})
	t.Run("failed to create spdy", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().Build()
		client := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeErrorInitNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForPod(ctx, readyPod, command, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create new spdy executor")
	})

	t.Run("failed to exec stream", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().Build()
		client := testclient.NewSimpleClientset(readyPod)
		sut := NewCommandExecutor(cli, client, &fake.RESTClient{})
		sut.commandExecutorCreator = fakeErrorStreamNewSPDYExecutor

		// when
		_, err := sut.ExecCommandForPod(ctx, readyPod, command, PodReady)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, assert.AnError.Error())
	})
}

func streamWithContextRun() func(ctx context.Context, options remotecommand.StreamOptions) error {
	return func(ctx context.Context, options remotecommand.StreamOptions) error {
		if options.Stdout != nil {
			buf := bytes.NewBufferString(commandOutput)
			if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestNewShellCommand(t *testing.T) {
	t.Run("should return simple command without args", func(t *testing.T) {
		actual := NewShellCommand("/bin/ls")

		expected := &shellCommand{command: "/bin/ls"}
		assert.Equal(t, expected, actual)
	})
	t.Run("should return command 1 arg", func(t *testing.T) {
		actual := NewShellCommand("/bin/ls", "/tmp/")

		expected := &shellCommand{command: "/bin/ls", args: []string{"/tmp/"}}
		assert.Equal(t, expected, actual)
	})
	t.Run("should return command multiple args", func(t *testing.T) {
		actual := NewShellCommand("/bin/ls", []string{"arg1", "arg2", "arg3"}...)

		expected := &shellCommand{command: "/bin/ls", args: []string{"arg1", "arg2", "arg3"}}
		assert.Equal(t, expected, actual)
	})
}

func TestShellCommand_String(t *testing.T) {
	type fields struct {
		Command string
		Args    []string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"return command", fields{"/bin/ls", nil}, "/bin/ls"},
		{"return command", fields{"/bin/ls", []string{}}, "/bin/ls"},
		{"return command and 1 arg", fields{"/bin/ls", []string{"/tmp"}}, "/bin/ls /tmp"},
		{"return command and multiple args", fields{"/bin/ls", []string{"/tmp", "/dir2"}}, "/bin/ls /tmp /dir2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewShellCommand(tt.fields.Command, tt.fields.Args...)
			assert.Equalf(t, tt.want, sc.String(), "String()")
		})
	}
}

func createFakeExecutors(t *testing.T) (a, b, c func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error)) {
	t.Helper()

	fakeNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		mockExecutor := newMockRemoteExecutor(t)
		mockExecutor.EXPECT().StreamWithContext(mock.Anything, mock.Anything).RunAndReturn(streamWithContextRun())
		return mockExecutor, nil
	}

	fakeErrorInitNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return nil, assert.AnError
	}

	fakeErrorStreamNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		mockExecutor := newMockRemoteExecutor(t)
		mockExecutor.EXPECT().StreamWithContext(mock.Anything, mock.Anything).Return(assert.AnError)
		return mockExecutor, nil
	}

	return fakeNewSPDYExecutor, fakeErrorInitNewSPDYExecutor, fakeErrorStreamNewSPDYExecutor
}

func TestNewShellCommandWithStdin(t *testing.T) {
	t.Run("should create shell command with stdin", func(t *testing.T) {
		// given
		stdin := strings.NewReader("abc")

		// when
		actual := NewShellCommandWithStdin(stdin, "base64")

		// then
		assert.Equal(t, &shellCommand{command: "base64", stdin: strings.NewReader("abc")}, actual)
	})
}
