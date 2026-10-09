package deletion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	testFinalizerNotFoundErr = apierrors.NewNotFound(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogus"}, testDoguName)
)

func TestRemoveFinalizerStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, *v3beta1.Dogu)
		want     stepsv3.StepResult
		assertFn func(t *testing.T, c K8sClient)
	}{
		{
			name: "should continue without update when finalizer is absent",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				doguResource.Finalizers = nil
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							t.Error("Update must not be called when finalizer is absent")
							return nil
						},
					}).Build()
				return c, doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should remove finalizer and continue",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()
				return c, doguResource
			},
			want: stepsv3.Continue(),
			assertFn: func(t *testing.T, c K8sClient) {
				doguResource := &v3beta1.Dogu{}
				require.NoError(t, c.Get(testCtx, testNamespacedName, doguResource))
				assert.False(t, controllerutil.ContainsFinalizer(doguResource, stepsv3.FinalizerName))
			},
		},
		{
			name: "should continue when dogu is already gone",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return testFinalizerNotFoundErr
						},
					}).Build()
				return c, doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should requeue with error when update fails",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return assert.AnError
						},
					}).Build()
				return c, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to remove finalizer from dogu %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, doguResource := tt.setup(t)
			sut := NewRemoveFinalizerStep(c)

			if got := sut.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, c)
			}
		})
	}
}

func TestNewRemoveFinalizerStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)

	sut := NewRemoveFinalizerStep(clientMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
}
