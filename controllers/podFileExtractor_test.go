package controllers

import (
	"context"
	_ "embed"
	"io"
	"net/url"
	"testing"

	utilmocks "github.com/cloudogu/k8s-dogu-operator/controllers/util/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	testing2 "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/remotecommand"
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
			k8sClient:   fakeClient,
			clientSet:   clientset,
			podExecutor: mockedPodExecutor,
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
			k8sClient:   fakeClient,
			clientSet:   clientset,
			podExecutor: mockedPodExecutor,
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
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("No such file or directory", nil)
		execPod := utilmocks.NewExecPod(t)
		execPod.On("ObjectKey").Return(ldapExecPodKey)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			podExecutor: mockedPodExecutor,
		}

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, execPod)

		// then
		require.NoError(t, err)
		assert.Empty(t, actual)
		assert.NotNil(t, actual)
		mockedPodExecutor.AssertExpectations(t)
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

func Test_newPodExec(t *testing.T) {
	t.Run("should return valid object", func(t *testing.T) {
		// when
		actual := newExecPod(&rest.Config{}, fake2.NewSimpleClientset(), testLdapExecPodKey)

		// then
		assert.NotEmpty(t, actual)
	})
}

func Test_podExec_execCmd(t *testing.T) {
	t.Run("should run command with error on failed container", func(t *testing.T) {
		podSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testLdapPodContainerName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  testLdapPodContainerName,
						Image: "official/ldap:1.2.3",
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodFailed},
		}

		clientset := fake2.NewSimpleClientset(podSpec)
		clientset.AddReactor("get", "v1/Pod", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
			return true, podSpec, nil
		})
		sut := newExecPod(&rest.Config{}, clientset, testLdapExecPodKey)

		// when
		_, errOut, err := sut.execCmd([]string{"/bin/ls", "/k8s/"})

		// then
		require.Error(t, err)
		assert.Empty(t, errOut)
		assert.Contains(t, err.Error(), "current phase is Failed")
	})
	t.Run("should run successfully", func(t *testing.T) {
		podSpec := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      testLdapPodContainerName,
				Namespace: testNamespace,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  testLdapPodContainerName,
						Image: "official/ldap:1.2.3",
					},
				},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		}

		clientset := fake2.NewSimpleClientset(podSpec)
		clientset.AddReactor("get", "v1/Pod", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
			return true, podSpec, nil
		})
		sut := newExecPod(&rest.Config{}, clientset, testLdapExecPodKey)
		mockedRestExecutor := &mockRestExecutor{}
		mockedRestExecutor.On("Execute").Return(nil)
		sut.restExecutor = mockedRestExecutor

		// when
		_, errOut, err := sut.execCmd([]string{"/bin/ls", "/k8s/"})

		// then
		require.NoError(t, err)
		assert.Empty(t, errOut)
		mockedRestExecutor.AssertExpectations(t)
	})
}

func Test_defaultPodExecutor_exec(t *testing.T) {
	t.Run("should fail with arbitrary error", func(t *testing.T) {
		sut := &defaultPodExecutor{&rest.Config{}, fake2.NewSimpleClientset()}

		// when
		actual, err := sut.exec(testLdapExecPodKey, "/bin/false")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not enumerate K8s resources in ExecPod ldap-execpod-1q2w3e")
		assert.Empty(t, actual)
	})
}

func newObjectKey(namespace, name string) *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
}

type mockPodExecutor struct {
	mock.Mock
}

func (m *mockPodExecutor) exec(podExecKey *client.ObjectKey, cmdArgs ...string) (stdOut string, err error) {
	args := m.Called(podExecKey, cmdArgs)
	return args.String(0), args.Error(1)
}

type mockRestExecutor struct {
	mock.Mock
}

func (m *mockRestExecutor) Execute(string, *url.URL, *rest.Config, io.Reader, io.Writer, io.Writer, bool, remotecommand.TerminalSizeQueue) error {
	args := m.Called()
	return args.Error(0)
}
