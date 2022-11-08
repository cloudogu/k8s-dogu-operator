package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	regmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	controllermocks "github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	resourceMocks "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
)

const namespace = "test"

type doguSupportManagerWithMocks struct {
	supportManager   *doguSupportManager
	doguRegistryMock *regmocks.DoguRegistry
	k8sClient        client.WithWatch
	recorderMock     *controllermocks.EventRecorder
}

func (d *doguSupportManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.doguRegistryMock,
		d.recorderMock,
	)
}

func getDoguSupportManagerWithMocks(scheme *runtime.Scheme) doguSupportManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	limitPatcher := &resourceMocks.LimitPatcher{}
	limitPatcher.On("RetrievePodLimits", mock.Anything).Return(limit.DoguLimits{}, nil)
	limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
	resourceGenerator := resource.NewResourceGenerator(scheme, limitPatcher)
	doguRegistry := &regmocks.DoguRegistry{}
	eventRecorder := &controllermocks.EventRecorder{}

	doguSupportManager := &doguSupportManager{
		client:            k8sClient,
		resourceGenerator: resourceGenerator,
		eventRecorder:     eventRecorder,
		doguRegistry:      doguRegistry,
	}

	return doguSupportManagerWithMocks{
		supportManager:   doguSupportManager,
		k8sClient:        k8sClient,
		doguRegistryMock: doguRegistry,
		recorderMock:     eventRecorder,
	}
}

func TestNewDoguSupportManager(t *testing.T) {
	// given
	k8sClient := fake.NewClientBuilder().Build()
	cesRegistry := &regmocks.Registry{}
	doguRegistry := &regmocks.DoguRegistry{}
	cesRegistry.On("DoguRegistry").Return(doguRegistry)
	recorder := &controllermocks.EventRecorder{}

	// when
	manager := NewDoguSupportManager(k8sClient, cesRegistry, recorder)

	// then
	require.NotNil(t, manager)
	mock.AssertExpectationsForObjects(t, cesRegistry, doguRegistry, recorder)
}

func Test_doguSupportManager_supportModeChanged(t *testing.T) {
	// given
	type args struct {
		doguResource *k8sv1.Dogu
		active       bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"return false for already set true flag", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: true}}, true}, false},
		{"return true for flag being unset", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: true}}, false}, true},
		{"return false for already set false flag", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: false}}, false}, false},
		{"return true for newly set false flag", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: false}}, true}, true},
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
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.doguRegistryMock.On("Get", "ldap").Return(ldap, nil)
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)
		resourceVersion := deployment.ResourceVersion

		// when
		err = sut.supportManager.updateDeployment(testCtx, ldapCr, deployment)

		// then
		require.NoError(t, err)
		sut.AssertMocks(t)

		deployment = &appsv1.Deployment{}
		err = sut.k8sClient.Get(testCtx, ldapCr.GetObjectKey(), deployment)
		require.NoError(t, err)
		assert.Greater(t, deployment.ResourceVersion, resourceVersion)
		assert.Equal(t, *readLdapDoguExpectedPodTemplateSupportOn(t), deployment.Spec.Template)
	})

	t.Run("error getting dogu descriptor", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.doguRegistryMock.On("Get", "ldap").Return(nil, assert.AnError)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: namespace}}
		err := sut.supportManager.client.Create(testCtx, deployment)
		require.NoError(t, err)

		// when
		err = sut.supportManager.updateDeployment(testCtx, ldapCr, nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get dogu descriptor of dogu ldap")
		sut.AssertMocks(t)
	})

	t.Run("error updating deployment of dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.doguRegistryMock.On("Get", "ldap").Return(ldap, nil)

		// when
		err := sut.supportManager.updateDeployment(testCtx, ldapCr, &appsv1.Deployment{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update dogu deployment ldap")
		sut.AssertMocks(t)
	})
}

func Test_doguSupportManager_HandleSupportMode(t *testing.T) {
	t.Run("return true on support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldap := readDoguDescriptor(t, ldapDoguDescriptorBytes)
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.doguRegistryMock.On("Get", "ldap").Return(ldap, nil)
		sut.recorderMock.On("Eventf", ldapCr, "Normal", "Support", "Support flag changed to %t. Deployment updated.", true)

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
		assert.True(t, result)
		sut.AssertMocks(t)
	})

	t.Run("error getting deployment from dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)

		// when
		_, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment of dogu ldap")
		sut.AssertMocks(t)
	})

	t.Run("return false and no error when no deployment ist found", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		sut.recorderMock.On("Eventf", ldapCr, "Warning", "Support", "No deployment found for dogu %s", "ldap")

		// when
		result, err := sut.supportManager.HandleSupportMode(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.False(t, result)
		sut.AssertMocks(t)
	})

	t.Run("return false on no support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = false
		sut.recorderMock.On("Event", ldapCr, "Normal", "Support", "Support flag did not change. Do nothing.")

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
		sut.AssertMocks(t)
	})

	t.Run("error updating deployment", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readDoguCr(t, ldapCrBytes)
		ldapCr.Namespace = namespace
		ldapCr.Spec.SupportMode = true
		sut.doguRegistryMock.On("Get", "ldap").Return(nil, assert.AnError)

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
		sut.AssertMocks(t)
	})
}
