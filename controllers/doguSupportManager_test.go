package controllers

import (
	"context"
	regmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	controllermocks "github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	resourceMocks "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

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
		scheme:            scheme,
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
	client := fake.NewClientBuilder().Build()
	cesRegistry := &regmocks.Registry{}
	doguRegistry := &regmocks.DoguRegistry{}
	cesRegistry.On("DoguRegistry").Return(doguRegistry)
	recorder := &controllermocks.EventRecorder{}

	// when
	manager := NewDoguSupportManager(client, cesRegistry, recorder)

	// then
	require.NotNil(t, manager)
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
		{"True flag and active mode should return false", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: true}}, true}, false},
		{"True flag and inactive mode should return true", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: true}}, false}, true},
		{"False flag and active mode should return false", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: false}}, false}, false},
		{"False flag and inactive mode should return true", args{&k8sv1.Dogu{Spec: k8sv1.DoguSpec{SupportMode: false}}, true}, true},
	}
	// when then
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsm := &doguSupportManager{}
			assert.Equalf(t, tt.want, dsm.supportModeChanged(tt.args.doguResource, tt.args.active), "supportModeChanged(%v, %v)", tt.args.doguResource, tt.args.active)
		})
	}
}

func Test_doguSupportManager_getDoguContainers(t *testing.T) {
	t.Run("successfully get dogu container", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"
		labels := make(map[string]string)
		labels["dogu"] = "ldap"
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test", Labels: labels},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}
		err := sut.k8sClient.Create(context.TODO(), pod)
		require.NoError(t, err)

		// when
		containers, err := sut.supportManager.getDoguContainers(context.TODO(), ldapCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, 1, len(containers))
		sut.AssertMocks(t)
	})

	t.Run("failed to get pod list", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		ldapCr := readTestDataLdapCr(t)

		// when
		_, err := sut.supportManager.getDoguContainers(context.TODO(), ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get list of pods from dogu")
		sut.AssertMocks(t)
	})
}

func Test_doguSupportManager_isSupportModeActive(t *testing.T) {
	t.Run("return true if one container has the env var SUPPORT_MODE to true", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		containers := []corev1.Container{
			{Env: []corev1.EnvVar{{Name: "SUPPORT_MODE", Value: "true"}}},
		}

		// when
		result := sut.supportManager.isSupportModeActive(containers)

		// then
		assert.True(t, result)
		sut.AssertMocks(t)
	})

	t.Run("return false if no container has the env var SUPPORT_MODE to true", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		containers := []corev1.Container{{}}

		// when
		result := sut.supportManager.isSupportModeActive(containers)

		// then
		assert.False(t, result)
		sut.AssertMocks(t)
	})
}

func Test_doguSupportManager_updateDeployment(t *testing.T) {
	t.Run("successfully update deployment", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldap := readTestDataLdapDogu(t)
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"
		ldapCr.Spec.SupportMode = true
		sut.doguRegistryMock.On("Get", "ldap").Return(ldap, nil)
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test"}}
		err := sut.supportManager.client.Create(context.TODO(), deployment)
		require.NoError(t, err)
		resourceVersion := deployment.ResourceVersion

		// when
		err = sut.supportManager.updateDeployment(context.TODO(), ldapCr)

		// then
		require.NoError(t, err)
		sut.AssertMocks(t)

		deployment = &appsv1.Deployment{}
		err = sut.k8sClient.Get(context.TODO(), *ldapCr.GetObjectKey(), deployment)
		require.NoError(t, err)
		assert.Greater(t, deployment.ResourceVersion, resourceVersion)
	})

	t.Run("error getting deployment of dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"

		// when
		err := sut.supportManager.updateDeployment(context.TODO(), ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get deployment of dogu ldap")
		sut.AssertMocks(t)
	})

	t.Run("error getting dogu descriptor", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"
		sut.doguRegistryMock.On("Get", "ldap").Return(nil, assert.AnError)

		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test"}}
		err := sut.supportManager.client.Create(context.TODO(), deployment)
		require.NoError(t, err)

		// when
		err = sut.supportManager.updateDeployment(context.TODO(), ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu deskriptor of dogu ldap")
		sut.AssertMocks(t)
	})
}

func Test_doguSupportManager_HandleSupportFlag(t *testing.T) {
	t.Run("return true on support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"
		ldapCr.Spec.SupportMode = true
		ldap := readTestDataLdapDogu(t)
		sut.doguRegistryMock.On("Get", "ldap").Return(ldap, nil)
		sut.recorderMock.On("Eventf", ldapCr, "Normal", "Support", "Support flag changed to %t. Deployment updated.", true)

		// add containers
		labels := make(map[string]string)
		labels["dogu"] = "ldap"
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test", Labels: labels},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}
		err := sut.k8sClient.Create(context.TODO(), pod)
		require.NoError(t, err)

		// add deployment
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test"}}
		err = sut.supportManager.client.Create(context.TODO(), deployment)
		require.NoError(t, err)

		// when
		result, err := sut.supportManager.HandleSupportFlag(context.TODO(), ldapCr)

		// then
		require.NoError(t, err)
		assert.True(t, result)
		sut.AssertMocks(t)
	})

	t.Run("error getting pod list from dogu", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(runtime.NewScheme())
		ldapCr := readTestDataLdapCr(t)

		// when
		_, err := sut.supportManager.HandleSupportFlag(context.TODO(), ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get list of pods from dogu ldap")
		sut.AssertMocks(t)
	})

	t.Run("return false on no support mode change", func(t *testing.T) {
		// given
		sut := getDoguSupportManagerWithMocks(getTestScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Namespace = "test"
		ldapCr.Spec.SupportMode = false
		sut.recorderMock.On("Event", ldapCr, "Normal", "Support", "Support flag did not change. Do nothing.")

		// add containers
		labels := make(map[string]string)
		labels["dogu"] = "ldap"
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "test", Labels: labels},
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "official/ldap:2.4.48-4"}, {Image: "other:1.2.3"}}}}
		err := sut.k8sClient.Create(context.TODO(), pod)
		require.NoError(t, err)

		// when
		result, err := sut.supportManager.HandleSupportFlag(context.TODO(), ldapCr)

		// then
		require.NoError(t, err)
		assert.False(t, result)
		sut.AssertMocks(t)
	})

	t.Run("error updating deployment", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "PodList",
		}, &corev1.PodList{})

		sut := getDoguSupportManagerWithMocks(scheme)
		ldapCr := readTestDataLdapCr(t)
		ldapCr.Spec.SupportMode = true

		// when
		_, err := sut.supportManager.HandleSupportFlag(context.TODO(), ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get deployment of dogu ldap")
		sut.AssertMocks(t)
	})
}
