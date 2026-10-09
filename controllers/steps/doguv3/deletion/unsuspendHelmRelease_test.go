package deletion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var testUnsuspendConflictErr = apierrors.NewConflict(schema.GroupResource{Group: "helm.toolkit.fluxcd.io", Resource: "helmreleases"}, testDoguName, assert.AnError)

func TestUnsuspendHelmReleaseStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu)
		want     stepsv3.StepResult
		assertFn func(t *testing.T, c K8sClient)
	}{
		{
			name: "should continue when HelmRelease does not exist",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should continue without update when HelmRelease is not suspended",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Spec.Suspend = false
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							t.Error("Update must not be called when HelmRelease is not suspended")
							return nil
						},
					}).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should unsuspend and continue when HelmRelease is suspended",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Spec.Suspend = true
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: stepsv3.Continue(),
			assertFn: func(t *testing.T, c K8sClient) {
				release := &fluxhelm.HelmRelease{}
				require.NoError(t, c.Get(testCtx, testNamespacedName, release))
				assert.False(t, release.Spec.Suspend)
			},
		},
		{
			name: "should emit event and requeue with error when unsuspend update fails",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Spec.Suspend = true
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).
					WithInterceptorFuncs(interceptor.Funcs{
						Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
							return assert.AnError
						},
					}).Build()

				wrappedErr := fmt.Errorf("failed to unsuspend HelmRelease %q: %w", testDoguName, assert.AnError)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, wrappedErr.Error())
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to unsuspend HelmRelease %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
		},
		{
			name: "should emit event and requeue with error when get fails",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).
					WithInterceptorFuncs(interceptor.Funcs{
						Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
							if _, ok := obj.(*fluxhelm.HelmRelease); ok {
								return assert.AnError
							}
							return c.Get(ctx, key, obj, opts...)
						},
					}).Build()

				wrappedErr := fmt.Errorf("failed to get HelmRelease %q: %w", testDoguName, assert.AnError)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, wrappedErr.Error())
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to get HelmRelease %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder, doguResource := tt.setup(t)
			sut := NewUnsuspendHelmReleaseStep(c, recorder)

			if got := sut.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, c)
			}
		})
	}
}

func TestNewUnsuspendHelmReleaseStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)
	recorderMock := NewMockEventRecorder(t)

	sut := NewUnsuspendHelmReleaseStep(clientMock, recorderMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
	assert.Equal(t, recorderMock, sut.eventRecorder)
}
