package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

const namespace = "test"

type doguSupportManagerWithMocks struct {
	supportManager       *doguSupportManager
	localDoguFetcherMock *mockLocalDoguFetcher
	k8sClient            client.WithWatch
	recorderMock         *mockEventRecorder
	podTemplateGenerator *mockPodTemplateResourceGenerator
}

func getDoguSupportManagerWithMocks(t *testing.T, scheme *runtime.Scheme) doguSupportManagerWithMocks {
	t.Helper()

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	podTemplateGenerator := newMockPodTemplateResourceGenerator(t)
	localDoguFetcherMock := newMockLocalDoguFetcher(t)
	eventRecorder := newMockEventRecorder(t)

	doguSupportManager := &doguSupportManager{
		client:                       k8sClient,
		podTemplateResourceGenerator: podTemplateGenerator,
		eventRecorder:                eventRecorder,
		doguFetcher:                  localDoguFetcherMock,
	}

	return doguSupportManagerWithMocks{
		supportManager:       doguSupportManager,
		k8sClient:            k8sClient,
		localDoguFetcherMock: localDoguFetcherMock,
		recorderMock:         eventRecorder,
		podTemplateGenerator: podTemplateGenerator,
	}
}

func TestNewDoguSupportManager(t *testing.T) {
	// given
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfig = createTestRestConfig

	k8sClient := fake.NewClientBuilder().Build()
	fetcher := newMockLocalDoguFetcher(t)
	generator := newMockResourceGenerator(t)
	recorder := newMockEventRecorder(t)

	// when
	manager := NewDoguSupportManager(k8sClient, fetcher, generator, recorder)

	// then
	assert.Same(t, k8sClient, manager.(*doguSupportManager).client)
	assert.Same(t, fetcher, manager.(*doguSupportManager).doguFetcher)
	assert.Same(t, generator, manager.(*doguSupportManager).podTemplateResourceGenerator)
	assert.Same(t, recorder, manager.(*doguSupportManager).eventRecorder)
}

func Test_doguSupportManager_supportModeChanged(t *testing.T) {
	// given
	type args struct {
		doguResource *doguv2.Dogu
		active       bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"return false for already set true flag", args{&doguv2.Dogu{Spec: doguv2.DoguSpec{SupportMode: true}}, true}, false},
		{"return true for flag being unset", args{&doguv2.Dogu{Spec: doguv2.DoguSpec{SupportMode: true}}, false}, true},
		{"return false for already set false flag", args{&doguv2.Dogu{Spec: doguv2.DoguSpec{SupportMode: false}}, false}, false},
		{"return true for newly set false flag", args{&doguv2.Dogu{Spec: doguv2.DoguSpec{SupportMode: false}}, true}, true},
	}
	// when then
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, supportModeChanged(tt.args.doguResource, tt.args.active), "supportModeChanged(%v, %v)", tt.args.doguResource, tt.args.active)
		})
	}
}

func Test_doguSupportManager_isDeploymentInSupportMode(t *testing.T) {
	t.Run("return true if one container has the env var SUPPORT_MODE to true", func(t *testing.T) {
		// given
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Env: []corev1.EnvVar{{Name: "SUPPORT_MODE", Value: "true"}}}}}}}}

		// when
		result := isDeploymentInSupportMode(deployment)

		// then
		assert.True(t, result)
	})

	t.Run("return false if no container has the env var SUPPORT_MODE to true", func(t *testing.T) {
		// given
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace}}

		// when
		result := isDeploymentInSupportMode(deployment)

		// then
		assert.False(t, result)
	})
}

func Test_doguSupportManager_updateDeployment(t *testing.T) {
	t.Run("successfully update deployment", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldap, nil)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)
		resourceVersion := deployment.ResourceVersion

		podSpec := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Hostname: "ldap",
				Containers: []corev1.Container{
					{
						Name:  "ldap",
						Image: "registry.cloudogu.com/official/ldap:2.4.48-4",
						Env:   []corev1.EnvVar{}},
				},
			},
		}

		sut.podTemplateGenerator.EXPECT().GetPodTemplate(testCtx, ldapCr, ldap).Return(&podSpec, nil)

		// when
		err = sut.supportManager.updateDeployment(testCtx, ldapCr, deployment)

		// then
		require.NoError(t, err)

		deployment = &appsv1.Deployment{}
		err = sut.k8sClient.Get(testCtx, ldapCr.GetObjectKey(), deployment)
		require.NoError(t, err)
		assert.Greater(t, deployment.ResourceVersion, resourceVersion)
		expectedPodSpec := corev1.PodSpec{
			Hostname: "ldap",
			Containers: []corev1.Container{
				{
					Name:  "ldap",
					Image: "registry.cloudogu.com/official/ldap:2.4.48-4",
					Env: []corev1.EnvVar{{
						Name:  "SUPPORT_MODE",
						Value: "true",
					}},
					Command: []string{"/bin/sh", "-c", "--"},
					Args:    []string{"while true; do sleep 5; done;"},
				},
			},
		}
		assert.Equal(t, expectedPodSpec, deployment.Spec.Template.Spec)
	})

	t.Run("error getting dogu descriptor", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(nil, assert.AnError)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)

		// when
		err = sut.supportManager.updateDeployment(testCtx, ldapCr, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get dogu descriptor of dogu ldap")
	})

	t.Run("error updating deployment of dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldap, nil)
		podSpec := corev1.PodTemplateSpec{Spec: corev1.PodSpec{}}
		sut.podTemplateGenerator.EXPECT().GetPodTemplate(testCtx, ldapCr, ldap).Return(&podSpec, nil)

		// when
		err := sut.supportManager.updateDeployment(testCtx, ldapCr, &appsv1.Deployment{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update dogu deployment ldap")
	})
}

func Test_doguSupportManager_HandleSupportMode(t *testing.T) {
	t.Run("return true on support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldap, nil)
		sut.recorderMock.On("Eventf", ldapCr, "Normal", "Support", "Support flag changed to %t. Deployment updated.", true)

		podTemplateSpec := corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}

		sut.podTemplateGenerator.EXPECT().GetPodTemplate(testCtx, ldapCr, ldap).Return(&podTemplateSpec, nil)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace},
			Spec: appsv1.DeploymentSpec{
				Template: podTemplateSpec}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)

		// when
		result, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("error getting deployment from dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, runtime.NewScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)

		// when
		_, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment of dogu ldap")
	})

	t.Run("return false and no error when no deployment ist found", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.recorderMock.On("Eventf", ldapCr, "Warning", "Support", "No deployment found for dogu %s when checking support handler", "ldap")

		// when
		result, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("return false on no support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = false

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)

		// when
		result, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("error updating deployment", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(t, getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(nil, assert.AnError)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)

		// when
		_, err = sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get dogu descriptor of dogu ldap")
	})
}
