package deletion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxsource "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestDeleteOCIRepositoryStep_Run(t *testing.T) {
	waitResult := stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonDeleting, fmt.Sprintf("waiting for OCIRepository %q to be deleted", testDoguName))

	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu)
		want     stepsv3.StepResult
		assertFn func(t *testing.T, c K8sClient)
	}{
		{
			name: "should continue when OCIRepository is already gone",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should delete OCIRepository and requeue while it is still present",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				repository := newTestOCIRepository()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: waitResult,
			assertFn: func(t *testing.T, c K8sClient) {
				repository := &fluxsource.OCIRepository{}
				err := c.Get(testCtx, testNamespacedName, repository)
				assert.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "should requeue while OCIRepository is still being deleted",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				repository := newTestOCIRepository()
				repository.Finalizers = []string{fluxFinalizer}
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).Build()
				require.NoError(t, c.Delete(testCtx, repository))
				return c, NewMockEventRecorder(t), doguResource
			},
			want: waitResult,
			assertFn: func(t *testing.T, c K8sClient) {
				repository := &fluxsource.OCIRepository{}
				require.NoError(t, c.Get(testCtx, testNamespacedName, repository))
				assert.False(t, repository.GetDeletionTimestamp().IsZero())
			},
		},
		{
			name: "should emit event and requeue with error when delete fails",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				repository := newTestOCIRepository()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, repository).
					WithInterceptorFuncs(interceptor.Funcs{
						Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
							return assert.AnError
						},
					}).Build()

				wrappedErr := fmt.Errorf("failed to delete OCIRepository %q: %w", testDoguName, assert.AnError)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, wrappedErr.Error())
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to delete OCIRepository %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
		},
		{
			name: "should emit event and requeue with error when get fails",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
							if _, ok := obj.(*fluxsource.OCIRepository); ok {
								return assert.AnError
							}
							return c.Get(ctx, key, obj, opts...)
						},
					}).Build()

				wrappedErr := fmt.Errorf("failed to get OCIRepository %q: %w", testDoguName, assert.AnError)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, wrappedErr.Error())
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to get OCIRepository %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder, doguResource := tt.setup(t)
			sut := NewDeleteOCIRepositoryStep(c, recorder)

			if got := sut.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, c)
			}
		})
	}
}

func TestNewDeleteOCIRepositoryStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)
	recorderMock := NewMockEventRecorder(t)

	sut := NewDeleteOCIRepositoryStep(clientMock, recorderMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
	assert.Equal(t, recorderMock, sut.eventRecorder)
}
