package migration

import (
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCleanupV2ResourcesStep_Run(t *testing.T) {
	for _, test := range []struct {
		name       string
		setupMocks func(t *testing.T, dogu *v3beta1.Dogu) (client.Client, ResourceFinder, EventRecorder)
		want       stepsv3.StepResult
		assertFn   func(t *testing.T, k8sClient client.Client, dogu *v3beta1.Dogu)
	}{
		{
			name: "continue if no (more) resources found",
			setupMocks: func(t *testing.T, dogu *v3beta1.Dogu) (client.Client, ResourceFinder, EventRecorder) {
				k8sClient := fake.NewClientBuilder().
					WithScheme(testScheme(t)).
					WithObjects(dogu).
					WithStatusSubresource(dogu).
					Build()
				finder := NewMockResourceFinder(t)
				finder.EXPECT().FindV2Resources(mock.Anything, getReference(dogu)).Return([]client.Object{}, nil)
				recorder := NewMockEventRecorder(t)
				return k8sClient, finder, recorder
			},
			want: stepsv3.Continue(),
			assertFn: func(t *testing.T, k8sClient client.Client, dogu *v3beta1.Dogu) {
				ready := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionReady)
				assert.NotNil(t, ready)
				assert.Equal(t, v1.ConditionFalse, ready.Status)
				assert.Equal(t, v3beta1.ReasonMigrationPending, ready.Reason)
			},
		},
		{
			name: "requeue with error if finder returns an error",
			setupMocks: func(t *testing.T, dogu *v3beta1.Dogu) (client.Client, ResourceFinder, EventRecorder) {
				k8sClient := fake.NewClientBuilder().
					WithScheme(testScheme(t)).
					WithObjects(dogu).
					WithStatusSubresource(dogu).
					Build()
				finder := NewMockResourceFinder(t)
				finder.EXPECT().FindV2Resources(mock.Anything, getReference(dogu)).Return([]client.Object{}, fmt.Errorf("some error"))
				recorder := NewMockEventRecorder(t)
				return k8sClient, finder, recorder
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to find v2 resources: %w", fmt.Errorf("some error")), v3beta1.ReasonMigrationPending),
		},
		{
			name: "requeue if deletion fails",
			setupMocks: func(t *testing.T, dogu *v3beta1.Dogu) (client.Client, ResourceFinder, EventRecorder) {
				k8sClient := fake.NewClientBuilder().
					WithScheme(testScheme(t)).
					WithObjects(dogu).
					WithStatusSubresource(dogu).
					Build()
				finder := NewMockResourceFinder(t)
				additionalObject := &coreV1.ConfigMap{Name: "test-cm", Namespace: "test"}
				finder.EXPECT().FindV2Resources(mock.Anything, getReference(dogu)).Return([]client.Object{additionalObject}, nil)
				recorder := NewMockEventRecorder(t)
				return k8sClient, finder, recorder
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to delete v2 resource: %w", errors.NewNotFound(schema.GroupResource{Resource: "configmaps"}, "test-cm")), v3beta1.ReasonMigrationPending),
		},
		{
			name: "delete any found objects",
			setupMocks: func(t *testing.T, dogu *v3beta1.Dogu) (client.Client, ResourceFinder, EventRecorder) {
				additionalObject := &coreV1.ConfigMap{Name: "test-cm", Namespace: "test"}
				k8sClient := fake.NewClientBuilder().
					WithScheme(testScheme(t)).
					WithObjects(dogu, additionalObject).
					WithStatusSubresource(dogu).
					Build()
				finder := NewMockResourceFinder(t)
				finder.EXPECT().FindV2Resources(mock.Anything, getReference(dogu)).Return([]client.Object{additionalObject}, nil)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(dogu, coreV1.EventTypeNormal, v3beta1.ReasonMigrationPending, "Initiated deletion of 1 resources of the v2 installation")
				return k8sClient, finder, recorder
			},
			want: stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonMigrationPending, "Resources of v2 dogu are being deleted"),
			assertFn: func(t *testing.T, k8sClient client.Client, dogu *v3beta1.Dogu) {
				var cm coreV1.ConfigMap
				err := k8sClient.Get(testCtx, types.NamespacedName{Name: "test-cm", Namespace: "test"}, &cm)
				assert.Error(t, err)
				assert.True(t, errors.IsNotFound(err))
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dogu := testDogu()
			k8sClient, finder, recorder := test.setupMocks(t, dogu)
			step := NewCleanupV2ResourcesStep(k8sClient, finder, recorder)

			result := step.Run(testCtx, dogu)

			assert.Equal(t, test.want, result)
			if test.assertFn != nil {
				test.assertFn(t, k8sClient, dogu)
			}
		})
	}
}

func getReference(dogu *v3beta1.Dogu) coreV1.ObjectReference {
	return coreV1.ObjectReference{
		Kind:            "Dogu",
		APIVersion:      "k8s.cloudogu.com/v3beta1",
		Name:            dogu.Name,
		Namespace:       dogu.Namespace,
		ResourceVersion: dogu.ResourceVersion,
	}
}
