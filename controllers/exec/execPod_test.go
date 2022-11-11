package exec

import (
	"bytes"
	"context"
	"github.com/cloudogu/k8s-dogu-operator/internal"
	"testing"

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
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks"
)

const testNamespace = "ecosystem"
const podName = "test-execpod-123abc"
const containerName = "ldap"

var testCtx = context.TODO()

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

func Test_execPod_Create(t *testing.T) {
	ldapDogu := readLdapDogu(t)
	ldapDoguResource := readLdapDoguResource(t)

	t.Run("should fail on pod spec creation", func(t *testing.T) {
		// given
		failureLdapDoguResource := readLdapDoguResource(t)
		failureLdapDoguResource.Namespace = "namespace-causing-failure"

		fakeClient := fake.NewClientBuilder().
			Build()
		sut := &execPod{client: fakeClient, podName: podName, dogu: ldapDogu, doguResource: failureLdapDoguResource}

		// when
		err := sut.Create(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set controller reference to exec pod test-execpod-123abc")
	})
	t.Run("should fail on resource creation", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(assert.AnError).
			On("Scheme").Once().Return(getTestScheme())

		sut := &execPod{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.Create(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on failed pod", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodFailed
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPod{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.Create(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod-123abc to spawn")
		assert.ErrorContains(t, err, "quitting dogu installation because exec pod test-execpod-123abc failed with status Failed or did not come up in time")
	})
	t.Run("should fail on other pod status", func(t *testing.T) {
		// given
		originalMaxTries := maxTries
		maxTries = 1
		mockClient := mocks.NewClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodPending
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPod{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.Create(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod-123abc to spawn")
		assert.ErrorContains(t, err, "found exec pod test-execpod-123abc but with status phase Pending")
		maxTries = originalMaxTries
	})
	t.Run("should fail on unable to find pod", func(t *testing.T) {
		// given
		originalMaxTries := maxTries
		maxTries = 1
		mockClient := mocks.NewClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Return(assert.AnError)

		sut := &execPod{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.Create(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for exec pod test-execpod-123abc to spawn")
		maxTries = originalMaxTries
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		objectKey := client.ObjectKey{Namespace: testNamespace, Name: podName}
		clientGetFn := func(args mock.Arguments) {
			pod := args[2].(*corev1.Pod)
			pod.Status.Phase = corev1.PodRunning
		}
		mockClient.
			On("Create", context.Background(), mock.Anything).Once().Return(nil).
			On("Scheme").Once().Return(getTestScheme()).
			On("Get", context.Background(), objectKey, mock.Anything).Run(clientGetFn).Return(nil)

		sut := &execPod{client: mockClient, podName: podName, dogu: ldapDogu, doguResource: ldapDoguResource}

		// when
		err := sut.Create(context.Background())

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
	sut := &execPod{client: fakeClient, doguResource: ldapDoguResource, dogu: ldapDogu}

	t.Run("should create exec pod same name as container name", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testCtx, testNamespace, containerName)

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
		actual, err := sut.createPod(testCtx, testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Name, containerName)
		assert.Equal(t, actual.Spec.Containers[0].ImagePullPolicy, corev1.PullAlways)

		config.Stage = originalStage
	})

	t.Run("should create exec pod from dogu image", func(t *testing.T) {
		// when
		actual, err := sut.createPod(testCtx, testNamespace, containerName)

		// then
		require.NoError(t, err)
		require.Len(t, actual.Spec.Containers, 1)
		assert.Equal(t, actual.Spec.Containers[0].Image, ldapDogu.Image+":"+ldapDogu.Version)
		assert.Equal(t, actual.Spec.Containers[0].ImagePullPolicy, corev1.PullIfNotPresent)
	})

	t.Run("should fail to set controller reference", func(t *testing.T) {
		// when
		_, err := sut.createPod(testCtx, "namespace-causing-failure", containerName)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to set controller reference to exec pod ldap")
	})
}

func Test_execPod_Exec(t *testing.T) {
	runningExecPod := &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-execpod-123abc", Namespace: testNamespace},
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
		sut := &execPod{
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
		mockExec := mocks.NewCommandExecutor(t)
		outBuf := bytes.NewBufferString("")
		mockExec.On("ExecCommandForPod", testCtx, runningExecPod, cmd, internal.ContainersStarted).Return(outBuf, assert.AnError)
		sut := &execPod{
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
		assert.Empty(t, actualOut)
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
		mockExec := mocks.NewCommandExecutor(t)
		outBuf := bytes.NewBufferString("possibly some output goes here")
		mockExec.On("ExecCommandForPod", testCtx, runningExecPod, cmd, internal.ContainersStarted).Return(outBuf, nil)
		sut := &execPod{
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
		assert.Equal(t, "possibly some output goes here", actualOut)
	})
}

func Test_execPod_createVolumes(t *testing.T) {
	t.Run("should return no resources for an unexpected execPod", func(t *testing.T) {
		// given
		const PodVolumeModeUnexpected = 3
		sut := &execPod{volumeMode: PodVolumeModeUnexpected}

		// when
		actualMounts, actualVolumes := sut.createVolumes(testCtx)

		// then
		assert.Nil(t, actualMounts)
		assert.Nil(t, actualVolumes)
	})
	t.Run("should return no volume resources for an install execPod", func(t *testing.T) {
		// given
		sut := &execPod{volumeMode: internal.VolumeModeInstall}

		// when
		actualMounts, actualVolumes := sut.createVolumes(testCtx)

		// then
		assert.Nil(t, actualMounts)
		assert.Nil(t, actualVolumes)
	})
	t.Run("should return volume resources for an upgrade execPod", func(t *testing.T) {
		// given
		ldapDoguResource := readLdapDoguResource(t)
		sut := &execPod{volumeMode: internal.VolumeModeUpgrade, doguResource: ldapDoguResource}

		// when
		actualMounts, actualVolumes := sut.createVolumes(testCtx)

		// then
		assert.NotEmpty(t, actualMounts)
		assert.Equal(t, "ldap-reserved", actualMounts[0].Name)
		assert.Equal(t, "/tmp/dogu-reserved", actualMounts[0].MountPath)
		assert.False(t, actualMounts[0].ReadOnly)

		assert.NotEmpty(t, actualVolumes)
		assert.Equal(t, "ldap-reserved", actualVolumes[0].Name)
		assert.Equal(t, "ldap-reserved", actualVolumes[0].VolumeSource.PersistentVolumeClaim.ClaimName)
		assert.False(t, actualVolumes[0].VolumeSource.PersistentVolumeClaim.ReadOnly)
	})
}

func Test_execPod_Delete(t *testing.T) {
	t.Run("should fail on arbitrary error", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		mockClient.
			On("Delete", context.Background(), &corev1.Pod{}).Once().Return(assert.AnError)

		sut := &execPod{podName: podName, client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to delete execPod "+podName)
	})
	t.Run("should succeed on not-found-error because target state is already reached", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		mockClient.On("Delete", context.Background(), &corev1.Pod{}).Once().Return(
			&errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}},
		)

		sut := &execPod{podName: podName, client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.NoError(t, err)
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		mockClient := mocks.NewClient(t)
		mockClient.
			On("Delete", context.Background(), &corev1.Pod{}).Once().Return(nil)

		sut := &execPod{client: mockClient, deleteSpec: &corev1.Pod{}}

		// when
		err := sut.Delete(context.Background())

		// then
		require.NoError(t, err)
	})
}

func Test_execPod_PodName(t *testing.T) {
	t.Run("should return podName", func(t *testing.T) {
		// given
		sut := &execPod{podName: podName}

		// when
		actual := sut.PodName()

		// then
		assert.Equal(t, podName, actual)
	})
}

func Test_generatePodName(t *testing.T) {
	suffixGen := mocks.NewSuffixGenerator(t)
	suffixGen.On("String", 6).Return("abc123")
	dogu := &core.Dogu{Name: "official/ldap"}

	actual := generatePodName(dogu, suffixGen)

	assert.Equal(t, "ldap-execpod-abc123", actual)
}

func TestNewExecPodFactory(t *testing.T) {
	actual := NewExecPodFactory(nil, nil, nil)
	assert.NotNil(t, actual)
}

func Test_defaultExecPodFactory_NewExecPod(t *testing.T) {
	suffixGen := mocks.NewSuffixGenerator(t)
	suffixGen.On("String", 6).Return("abc123")
	fakeClient := fake.NewClientBuilder().
		WithScheme(getTestScheme()).
		Build()
	clientSet := fake2.NewSimpleClientset()
	restConfig := &rest.Config{}
	commandExec := NewCommandExecutor(fakeClient, clientSet, clientSet.CoreV1().RESTClient())
	dogu := &core.Dogu{Name: "official/ldap"}

	sut := NewExecPodFactory(fakeClient, restConfig, commandExec)
	sut.suffixGen = suffixGen

	// when
	pod, err := sut.NewExecPod(internal.VolumeModeInstall, nil, dogu)

	// then
	require.NoError(t, err)
	assert.NotNil(t, pod)
}
