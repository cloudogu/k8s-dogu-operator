package controllers

import (
	"context"
	_ "embed"
	"io"
	"net/url"
	"testing"

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

func Test_podFileExtractor_createExecPodSpec(t *testing.T) {
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	fakeClient := fake.NewClientBuilder().
		WithScheme(getTestScheme()).
		Build()
	sut := &podFileExtractor{
		k8sClient: fakeClient,
		suffixGen: &testSuffixGenerator{},
	}

	t.Run("should create exec container name with pseudo-unique suffix", func(t *testing.T) {
		// when
		_, containerName, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, testLdapPodContainerName, containerName)
	})

	t.Run("should create exec pod same name as container name", func(t *testing.T) {
		// when
		podspec, containerName, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		require.Len(t, podspec.Spec.Containers, 1)
		assert.Equal(t, podspec.Spec.Containers[0].Name, containerName)
	})

	t.Run("should create exec pod from dogu image", func(t *testing.T) {
		// when
		podspec, _, err := sut.createExecPodSpec(testNamespace, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		require.Len(t, podspec.Spec.Containers, 1)
		assert.Equal(t, podspec.Spec.Containers[0].Image, ldapDogu.Image+":"+ldapDogu.Version)
	})
}

func Test_defaultPodFinder_find(t *testing.T) {
	t.Run("should find running pod immediately", func(t *testing.T) {
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
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(podSpec).
			Build()
		sut := &defaultPodFinder{k8sClient: fakeClient}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		err := sut.find(testContext, testLdapExecPodKey)

		// then
		require.NoError(t, err)
	})
	t.Run("should return expressive error for unready pod after timeout", func(t *testing.T) {
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
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(podSpec).
			Build()
		sut := &defaultPodFinder{k8sClient: fakeClient}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		err := sut.find(testContext, testLdapExecPodKey)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "did not come up in time")
		assert.Contains(t, err.Error(), testLdapPodContainerName)
		assert.Contains(t, err.Error(), "status Failed")
	})
	t.Run("should return expressive error for non-existing pod", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			// No PodSpec here
			Build()
		sut := &defaultPodFinder{k8sClient: fakeClient}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		err := sut.find(testContext, testLdapExecPodKey)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ldap-execpod-1q2w3e could not be found")
	})
}

func Test_podFileExtractor_ExtractK8sResourcesFromContainer(t *testing.T) {
	ldapCr := readDoguCr(t, ldapCrBytes)
	// simulate dogu in a non-default namespace
	ldapCr.Namespace = testNamespace
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)

	t.Run("should fail with non-existing exec pod", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", testLdapExecPodKey).Return(assert.AnError)
		mockedPodExecutor := &mockPodExecutor{}

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should fail with command error on exec pod", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", testLdapExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("", assert.AnError)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.Nil(t, actual)
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should run successfully with file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", testLdapExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("test-k8s-resources.yaml", nil)
		expectedCatCommand := []string{"/bin/cat", "/k8s/test-k8s-resources.yaml"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedCatCommand).Return("resource { content : goes-here }", nil)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		expectedFileMap := make(map[string]string)
		expectedFileMap["/k8s/test-k8s-resources.yaml"] = "resource { content : goes-here }"
		assert.Equal(t, expectedFileMap, actual)
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should run successfully without file output", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", testLdapExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedLsCommand := []string{"/bin/bash", "-c", "/bin/ls /k8s/ || true"}
		mockedPodExecutor.On("exec", testLdapExecPodKey, expectedLsCommand).Return("No such file or directory", nil)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		actual, err := sut.ExtractK8sResourcesFromContainer(testContext, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		assert.Empty(t, actual)
		assert.NotNil(t, actual)
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})

}

func Test_podFileExtractor_ExtractScriptResourcesFromContainer(t *testing.T) {
	redmineCr := readDoguCr(t, redmineCrBytes)
	// simulate dogu in a non-default namespace
	redmineCr.Namespace = testNamespace
	redmineDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
	redminePodContainerName := "redmine-execpod-" + testPodContainerNameSuffix
	redmineExecPodKey := newObjectKey(testNamespace, redminePodContainerName)

	t.Run("should return found script", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", redmineExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedCatCommand := []string{"/bin/bash", "-c", "/bin/cat", "/pre-upgrade.sh"}
		scriptContent := "#!/bin/bash\necho hello world"
		mockedPodExecutor.On("exec", redmineExecPodKey, expectedCatCommand).Return(scriptContent, nil)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		actual, err := sut.ExtractScriptResourcesFromContainer(testContext, redmineCr, redmineDogu, "pre-upgrade")

		// then
		require.NoError(t, err)
		expected := map[string]string{"/pre-upgrade.sh": scriptContent}
		assert.Equal(t, expected, actual)
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should return empty map on no pre-upgrade script", func(t *testing.T) {
		// given
		redmineDogu := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDogu.ExposedCommands = nil
		sut := &podFileExtractor{}

		// when
		actual, err := sut.ExtractScriptResourcesFromContainer(nil, redmineCr, redmineDogu, "pre-upgrade")

		// then
		require.NoError(t, err)
		assert.Equal(t, map[string]string{}, actual)
	})
	t.Run("should fail on script error", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", redmineExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedCatCommand := []string{"/bin/bash", "-c", "/bin/cat", "/pre-upgrade.sh"}
		mockedPodExecutor.On("exec", redmineExecPodKey, expectedCatCommand).Return("file not found", assert.AnError)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		_, err := sut.ExtractScriptResourcesFromContainer(testContext, redmineCr, redmineDogu, "pre-upgrade")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "error while getting file /pre-upgrade.sh")
		assert.Contains(t, err.Error(), "file not found")
		mockedPodFinder.AssertExpectations(t)
		mockedPodExecutor.AssertExpectations(t)
	})
	t.Run("should fail on missing script", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		clientset := fake2.NewSimpleClientset()
		mockedPodFinder := &mockPodFinder{}
		mockedPodFinder.On("find", redmineExecPodKey).Return(nil)
		mockedPodExecutor := &mockPodExecutor{}
		expectedCatCommand := []string{"/bin/bash", "-c", "/bin/cat", "/pre-upgrade.sh"}
		mockedPodExecutor.On("exec", redmineExecPodKey, expectedCatCommand).Return("No such file or directory", nil)

		sut := &podFileExtractor{
			k8sClient:   fakeClient,
			clientSet:   clientset,
			suffixGen:   &testSuffixGenerator{},
			podFinder:   mockedPodFinder,
			podExecutor: mockedPodExecutor,
		}
		// decrease waiting time; must not be lower than 2
		maxTries = 2
		defer func() { maxTries = 20 }()

		// when
		_, err := sut.ExtractScriptResourcesFromContainer(testContext, redmineCr, redmineDogu, "pre-upgrade")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not find exposed command /pre-upgrade.sh")
		mockedPodFinder.AssertExpectations(t)
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

func Test_createPodExecObjectKey(t *testing.T) {
	const podName = "le-test-pod-name"

	actual := createExecPodObjectKey(testNamespace, podName)

	assert.NotEmpty(t, actual)
	assert.Equal(t, newObjectKey(testNamespace, podName), actual)
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
		assert.Contains(t, err.Error(), "could not enumerate K8s resources in execPod ldap-execpod-1q2w3e")
		assert.Empty(t, actual)
	})
}

func Test_defaultSufficeGenerator_String(t *testing.T) {
	actual := (&defaultSufficeGenerator{}).String(6)
	assert.Len(t, actual, 6)
}

func newObjectKey(namespace, name string) *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
}

type testSuffixGenerator struct{}

func (t *testSuffixGenerator) String(_ int) string {
	return testPodContainerNameSuffix
}

type mockPodFinder struct {
	mock.Mock
}

func (m *mockPodFinder) find(_ context.Context, podExecKey *client.ObjectKey) error {
	args := m.Called(podExecKey)
	return args.Error(0)
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
