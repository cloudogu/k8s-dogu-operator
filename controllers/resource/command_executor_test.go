package resource

import (
	"bytes"
	"context"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

type fakeExecutor struct {
	method string
	url    *url.URL
}

type fakeFailExecutor struct {
	method string
	url    *url.URL
}

func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := bytes.NewBufferString("username:user")
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeFailExecutor) Stream(_ remotecommand.StreamOptions) error {
	return assert.AnError
}

func TestExposedCommandExecutor_ExecCommand(t *testing.T) {
	ctx := context.TODO()
	labels := map[string]string{}
	labels["dogu"] = "postgresql"
	readyPod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}, Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady}}}}
	unreadyPod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
	command := &core.ExposedCommand{
		Name:        "create-sa-command",
		Description: "desc",
		Command:     "/create-sa.sh",
	}
	params := []string{"ro"}

	fakeNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeExecutor{method: method, url: url}, nil
	}
	fakeErrorInitNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return nil, assert.AnError
	}
	fakeErrorStreamNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeFailExecutor{method: method, url: url}, nil
	}

	oldConfigFunc := config.GetConfigOrDie
	ctrl.GetConfigOrDie = func() *rest.Config {
		return nil
	}
	defer func() {
		ctrl.GetConfigOrDie = oldConfigFunc
	}()

	t.Run("success", func(t *testing.T) {
		// given
		client := testclient.NewSimpleClientset(&readyPod)
		commandExecutor := NewCommandExecutor(client, &fake.RESTClient{})
		commandExecutor.CommandExecutorCreator = fakeNewSPDYExecutor
		expectedBuffer := bytes.NewBufferString("username:user")

		// when
		buffer, err := commandExecutor.ExecCommand(ctx, "postgresql", "test", command, params)

		// then
		require.NoError(t, err)
		require.NotNil(t, buffer)
		assert.Equal(t, expectedBuffer, buffer)
	})

	t.Run("found no pods", func(t *testing.T) {
		// given
		client := testclient.NewSimpleClientset()
		commandExecutor := NewCommandExecutor(client, &fake.RESTClient{})
		commandExecutor.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := commandExecutor.ExecCommand(ctx, "postgresql", "test", nil, nil)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "found no pods for dogu postgresql")
	})

	t.Run("pod is not ready", func(t *testing.T) {
		// given
		client := testclient.NewSimpleClientset(&unreadyPod)
		commandExecutor := NewCommandExecutor(client, &fake.RESTClient{})
		commandExecutor.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		_, err := commandExecutor.ExecCommand(ctx, "postgresql", "test", nil, nil)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "can't execute command in pod with status")
	})

	t.Run("failed to create spdy", func(t *testing.T) {
		// given
		client := testclient.NewSimpleClientset(&readyPod)
		commandExecutor := NewCommandExecutor(client, &fake.RESTClient{})
		commandExecutor.CommandExecutorCreator = fakeErrorInitNewSPDYExecutor

		// when
		_, err := commandExecutor.ExecCommand(ctx, "postgresql", "test", command, params)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create new spdy executor")
	})

	t.Run("failed to exec stream", func(t *testing.T) {
		// given
		client := testclient.NewSimpleClientset(&readyPod)
		commandExecutor := NewCommandExecutor(client, &fake.RESTClient{})
		commandExecutor.CommandExecutorCreator = fakeErrorStreamNewSPDYExecutor

		// when
		_, err := commandExecutor.ExecCommand(ctx, "postgresql", "test", command, params)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), assert.AnError.Error())
	})
}
