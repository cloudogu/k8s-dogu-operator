package controllers

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguAdditionalIngressAnnotationsManager(t *testing.T) {
	// when
	manager := NewDoguAdditionalIngressAnnotationsManager(fake.NewClientBuilder().Build(), newMockEventRecorder(t))

	// then
	require.NotNil(t, manager)
}

func Test_doguAdditionalIngressAnnotationsManager_SetDoguAdditionalIngressAnnotations(t *testing.T) {
	t.Run("success without annotations", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: dogu.Name, Namespace: dogu.Namespace}}
		client := fake.NewClientBuilder().WithObjects(doguService).Build()
		sut := NewDoguAdditionalIngressAnnotationsManager(client, newMockEventRecorder(t))

		// when
		err := sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("success with annotations", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		ingressAnnotation := map[string]string{"test": "test"}
		marshal, err := json.Marshal(ingressAnnotation)
		require.NoError(t, err)
		annotations := map[string]string{
			"k8s-dogu-operator.cloudogu.com/additional-ingress-annotations": string(marshal),
		}
		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: dogu.Name, Namespace: dogu.Namespace, Annotations: annotations}}
		client := fake.NewClientBuilder().WithObjects(doguService).Build()
		sut := NewDoguAdditionalIngressAnnotationsManager(client, newMockEventRecorder(t))

		// when
		err = sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("should throw error if no service is found", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		sut := NewDoguAdditionalIngressAnnotationsManager(client, newMockEventRecorder(t))

		// when
		err := sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch service for dogu 'ldap'")
	})

	t.Run("should fail to update service", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		ingressAnnotation := map[string]string{"test": "test"}
		marshal, err := json.Marshal(ingressAnnotation)
		require.NoError(t, err)
		annotations := map[string]string{
			"k8s-dogu-operator.cloudogu.com/additional-ingress-annotations": string(marshal),
		}
		doguService := v1.Service{ObjectMeta: metav1.ObjectMeta{Name: dogu.Name, Namespace: dogu.Namespace, Annotations: annotations}}
		client := NewMockK8sClient(t)
		client.EXPECT().Get(context.TODO(), dogu.GetObjectKey(), mock.AnythingOfType("*v1.Service")).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object k8s.Object, option ...k8s.GetOption) error {
			servicePtr := object.(*v1.Service)
			*servicePtr = doguService
			return nil
		})
		client.EXPECT().Update(context.TODO(), mock.AnythingOfType("*v1.Service")).Return(assert.AnError)
		sut := NewDoguAdditionalIngressAnnotationsManager(client, newMockEventRecorder(t))

		// when
		err = sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu service 'ldap' with ingress annotations")
	})
}
