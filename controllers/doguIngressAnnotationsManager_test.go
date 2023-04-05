package controllers

import (
	"context"
	"encoding/json"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguAdditionalIngressAnnotationsManager(t *testing.T) {
	// when
	manager := NewDoguAdditionalIngressAnnotationsManager(fake.NewClientBuilder().Build(), mocks.NewEventRecorder(t))

	// then
	require.NotNil(t, manager)
}

func Test_doguAdditionalIngressAnnotationsManager_SetDoguAdditionalIngressAnnotations(t *testing.T) {
	t.Run("success without annotations", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		doguService := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: dogu.Name, Namespace: dogu.Namespace}}
		client := fake.NewClientBuilder().WithObjects(doguService).Build()
		sut := NewDoguAdditionalIngressAnnotationsManager(client, mocks.NewEventRecorder(t))

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
		sut := NewDoguAdditionalIngressAnnotationsManager(client, mocks.NewEventRecorder(t))

		// when
		err = sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("should throw error if no service is found", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		sut := NewDoguAdditionalIngressAnnotationsManager(client, mocks.NewEventRecorder(t))

		// when
		err := sut.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch service for dogu 'ldap'")
	})
}
