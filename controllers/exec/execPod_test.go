package exec

import (
	"bytes"
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

const testNamespace = "ecosystem"
const podName = "test-execpod"
const containerName = "ldap"

var testCtx = context.TODO()

func Test_execPod_Create(t *testing.T) {
	ldapDogu := readLdapDogu(t)
	ldapDoguResource := readLdapDoguResource(t)

	t.Run("should fail on pod spec creation", func(t *testing.T) {
		// given
		failureLdapDoguResource := readLdapDoguResource(t)
		failureLdapDoguResource.Namespace = "namespace-causing-failure"

		fakeClient := fake.NewClientBuilder().
			Build()
		sut := &execPodFactory{client: fakeClient, podName: podName, dogu: ldapDogu, doguResource: failureLdapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set controller reference to exec pod test-execpod")
	})
	t.Run("should fail on resource creation", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(assert.AnError).
			On("Scheme").Once().Return(getTestScheme())

		sut := &execPodFactory{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on failed pod", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodFailed
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPodFactory{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod to spawn")
		assert.ErrorContains(t, err, "quitting dogu installation because exec pod test-execpod failed with status Failed or did not come up in time")
	})
	t.Run("should fail on other pod status", func(t *testing.T) {
		// given
		originalMaxWaitDuration := maxWaitDuration
		maxWaitDuration = time.Second * 3
		mockClient := newMockK8sClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodPending
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPodFactory{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod to spawn")
		assert.ErrorContains(t, err, "found exec pod test-execpod but with status phase Pending")
		maxWaitDuration = originalMaxWaitDuration
	})
	t.Run("should fail on unable to find pod", func(t *testing.T) {
		// given
		originalMaxWaitDuration := maxWaitDuration
		maxWaitDuration = time.Second * 3
		mockClient := newMockK8sClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Return(assert.AnError)

		sut := &execPodFactory{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod to spawn")
		maxWaitDuration = originalMaxWaitDuration
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodRunning
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPodFactory{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.CreateBlocking(context.Background())

		// then
		require.NoError(t, err)
		assert.Equal(t, podName, sut.deleteSpec.ObjectMeta.Name)
		assert.Equal(t, "ecosystem", sut.deleteSpec.ObjectMeta.Namespace)
		assert.NotEmpty(t, sut.deleteSpec)
	})
}

func Test_execPod_createPod(t *testing.T) {
	ldapDogu := readLdapDogu(t)
	ldapDoguResource := readLdapDoguResource(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(getTestScheme()).
		Build()
	sut := &execPodFactory{client: fakeClient, doguResource: ldapDoguResource, dogu: ldapDogu}

	t.Run("should create exec pod same name as container name", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Name, containerName)
		assert.Equal(t, actual.Spec.Containers[0].ImagePullPolicy, corev1.PullIfNotPresent)
	})

	t.Run("should create exec pod same name as container name with stage development", func(t *testing.T) {
		// given
		originalStage := config.Stage
		config.Stage = config.StageDevelopment

		// when
		actual, err := sut.createPod(testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Name, containerName)
		assert.Equal(t, actual.Spec.Containers[0].ImagePullPolicy, corev1.PullAlways)

		config.Stage = originalStage
	})

	t.Run("should create exec pod from dogu image", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Image, ldapDogu.Image+":"+ldapDogu.Version)
		assert.Equal(t, actual.Spec.Containers[0].ImagePullPolicy, corev1.PullIfNotPresent)
	})

	t.Run("should fail to set controller reference", func(t *testing.T) {
		// when
		_, err := sut.createPod("namespace-causing-failure", containerName)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set controller reference to exec pod ldap")
	})
}

func Test_execPod_Exec(t *testing.T) {
	runningExecPod := &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-execpod", Namespace: testNamespace},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	t.Run("should fail when getting pod", func(t *testing.T) {
		// given
		ldapDogu := readLdapDogu(t)
		ldapDoguResource := readLdapDoguResource(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects().
			Build()
		cmd := &shellCommand{command: "/bin/ls", args: []string{"-lahF"}}
		sut := &execPodFactory{
			client:       fakeClient,
			doguResource: ldapDoguResource,
			dogu:         ldapDogu,
			podName:      podName,
		}

		// when
		actualOut, err := sut.Exec(testCtx, cmd)

		// then
		require.Error(t, err)
		require.ErrorContains(t, err, "could not get pod")
		assert.Empty(t, actualOut)
	})
	t.Run("should fail with error in commandExecutor", func(t *testing.T) {
		// given
		ldapDogu := readLdapDogu(t)
		ldapDoguResource := readLdapDoguResource(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(runningExecPod).
			Build()
		cmd := &shellCommand{command: "/bin/ls", args: []string{"-lahF"}}
		mockExec := NewMockCommandExecutor(t)
		outBuf := bytes.NewBufferString("")
		mockExec.On("ExecCommandForPod", testCtx, runningExecPod, cmd, ContainersStarted).Return(outBuf, assert.AnError)
		sut := &execPodFactory{
			client:       fakeClient,
			doguResource: ldapDoguResource,
			dogu:         ldapDogu,
			podName:      podName,
			executor:     mockExec,
		}

		// when
		actualOut, err := sut.Exec(testCtx, cmd)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Empty(t, actualOut.String())
	})
	t.Run("should be successful", func(t *testing.T) {
		// given
		ldapDogu := readLdapDogu(t)
		ldapDoguResource := readLdapDoguResource(t)
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(runningExecPod).
			Build()
		cmd := &shellCommand{command: "/bin/ls", args: []string{"-lahF"}}
		mockExec := NewMockCommandExecutor(t)
		outBuf := bytes.NewBufferString("possibly some output goes here")
		mockExec.On("ExecCommandForPod", testCtx, runningExecPod, cmd, ContainersStarted).Return(outBuf, nil)
		sut := &execPodFactory{
			client:       fakeClient,
			doguResource: ldapDoguResource,
			dogu:         ldapDogu,
			podName:      podName,
			executor:     mockExec,
		}

		// when
		actualOut, err := sut.Exec(testCtx, cmd)

		// then
		require.NoError(t, err)
		assert.Equal(t, "possibly some output goes here", actualOut.String())
	})
}

func Test_execPod_Delete(t *testing.T) {
	t.Run("should fail on arbitrary error", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		mockClient.
			On("Delete", context.Background(), &corev1.Pod{}).Once().Return(assert.AnError)

		sut := &execPodFactory{podName: podName, client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete execPodFactory "+podName)
	})
	t.Run("should succeed on not-found-error because target state is already reached", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		mockClient.On("Delete", context.Background(), &corev1.Pod{}).Once().Return(
			&errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}},
		)

		sut := &execPodFactory{podName: podName, client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockClient := newMockK8sClient(t)
		mockClient.
			On("Delete", context.Background(), &corev1.Pod{}).Once().Return(nil)

		sut := &execPodFactory{client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.NoError(t, err)
	})
}

func Test_execPod_PodName(t *testing.T) {
	t.Run("should return podName", func(t *testing.T) {
		// given
		sut := &execPodFactory{podName: podName}

		// when
		actual := sut.PodName()

		// then
		assert.Equal(t, podName, actual)
	})
}

func TestNewExecPodFactory(t *testing.T) {
	actual := NewExecPodFactory(nil, nil, nil)
	assert.NotNil(t, actual)
}

func Test_defaultExecPodFactory_NewExecPod(t *testing.T) {
	suffixGen := newMockSuffixGenerator(t)
	suffixGen.On("String", 6).Return("abc123")
	fakeClient := fake.NewClientBuilder().
		WithScheme(getTestScheme()).
		Build()
	clientSet := fake2.NewSimpleClientset()
	restConfig := &rest.Config{}
	commandExec := NewCommandExecutor(fakeClient, clientSet, clientSet.CoreV1().RESTClient())
	dogu := &core.Dogu{Name: "official/ldap"}

	sut := NewExecPodFactory(fakeClient, restConfig, commandExec)

	// when
	pod := sut.NewExecPod(nil, dogu)

	// then
	assert.NotNil(t, pod)
}
