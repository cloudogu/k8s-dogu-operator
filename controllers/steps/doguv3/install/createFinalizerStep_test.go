package install

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var testFinalizerConflictErr = apierrors.NewConflict(schema.GroupResource{Group: "k8s.cloudogu.com", Resource: "dogus"}, testDoguName, assert.AnError)

func TestCreateFinalizerStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, *v3beta1.Dogu)
		want     doguv3.StepResult
		assertFn func(t *testing.T, c K8sClient)
	}{
		{
			name: "should continue without update when finalizer already present",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				controllerutil.AddFinalizer(doguResource, doguv3.FinalizerName)
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							t.Error("Update must not be called when finalizer already present")
							return nil
						},
					}).Build()
				return c, doguResource
			},
			want: doguv3.Continue(),
		},
		{
			name: "should add finalizer and continue",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()
				return c, doguResource
			},
			want: doguv3.Continue(),
			assertFn: func(t *testing.T, c K8sClient) {
				doguResource := &v3beta1.Dogu{}
				require.NoError(t, c.Get(testCtx, testNamespacedName, doguResource))
				assert.True(t, controllerutil.ContainsFinalizer(doguResource, doguv3.FinalizerName))
			},
		},
		{
			name: "should requeue with error when update fails",
			setup: func(t *testing.T) (K8sClient, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return assert.AnError
						},
					}).Build()
				return c, doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to add finalizer to dogu %q: %w", testDoguName, assert.AnError), v3beta1.ReasonInstalling),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, doguResource := tt.setup(t)
			sut := NewCreateFinalizerStep(c)

			if got := sut.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, c)
			}
		})
	}
}

func TestNewCreateFinalizerStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)

	sut := NewCreateFinalizerStep(clientMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
}
