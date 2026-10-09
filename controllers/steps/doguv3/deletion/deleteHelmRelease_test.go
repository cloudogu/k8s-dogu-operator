package deletion

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxhelm "github.com/fluxcd/helm-controller/api/v2"
	fluxmeta "github.com/fluxcd/pkg/apis/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestDeleteHelmReleaseStep_Run(t *testing.T) {
	waitResult := stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonDeleting, fmt.Sprintf("waiting for HelmRelease %q to be deleted", testDoguName))

	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu)
		want     stepsv3.StepResult
		assertFn func(t *testing.T, c K8sClient)
	}{
		{
			name: "should continue when HelmRelease is already gone",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: stepsv3.Continue(),
		},
		{
			name: "should delete HelmRelease and requeue while it is still present",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).Build()
				return c, NewMockEventRecorder(t), doguResource
			},
			want: waitResult,
			assertFn: func(t *testing.T, c K8sClient) {
				release := &fluxhelm.HelmRelease{}
				err := c.Get(testCtx, testNamespacedName, release)
				assert.True(t, apierrors.IsNotFound(err))
			},
		},
		{
			name: "should requeue while HelmRelease is still being uninstalled",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Finalizers = []string{fluxFinalizer}
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).Build()
				// Mark the release as deleting; the flux finalizer keeps it around while flux uninstalls.
				require.NoError(t, c.Delete(testCtx, release))
				return c, NewMockEventRecorder(t), doguResource
			},
			want: waitResult,
			assertFn: func(t *testing.T, c K8sClient) {
				release := &fluxhelm.HelmRelease{}
				require.NoError(t, c.Get(testCtx, testNamespacedName, release))
				assert.False(t, release.GetDeletionTimestamp().IsZero())
			},
		},
		{
			name: "should escalate but keep requeueing when HelmRelease uninstall is stalled",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Finalizers = []string{fluxFinalizer}
				release.Status.Conditions = []metav1.Condition{{
					Type:    fluxmeta.StalledCondition,
					Status:  metav1.ConditionTrue,
					Reason:  "RetriesExceeded",
					Message: "uninstall retries exhausted",
				}}
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).Build()
				// Let it linger with a deletion timestamp, mirroring a stuck flux uninstall.
				require.NoError(t, c.Delete(testCtx, release))

				msg := fmt.Sprintf("HelmRelease %q uninstall stalled: %s", testDoguName, "uninstall retries exhausted")
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionStalled, msg)
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueAfter(defaultRequeueAfter, stepsv3.ReasonDeletionStalled, fmt.Sprintf("HelmRelease %q uninstall stalled: %s", testDoguName, "uninstall retries exhausted")),
		},
		{
			name: "should escalate but keep requeueing when HelmRelease reports uninstall failed",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				release.Finalizers = []string{fluxFinalizer}
				release.Status.Conditions = []metav1.Condition{{
					Type:    fluxhelm.ReleasedCondition,
					Status:  metav1.ConditionFalse,
					Reason:  fluxhelm.UninstallFailedReason,
					Message: "helm uninstall failed",
				}}
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).Build()
				require.NoError(t, c.Delete(testCtx, release))

				msg := fmt.Sprintf("HelmRelease %q uninstall failed: %s", testDoguName, "helm uninstall failed")
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionStalled, msg)
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueAfter(defaultRequeueAfter, stepsv3.ReasonDeletionStalled, fmt.Sprintf("HelmRelease %q uninstall failed: %s", testDoguName, "helm uninstall failed")),
		},
		{
			name: "should emit event and requeue with error when delete fails",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := newTestDoguResource()
				release := newTestHelmRelease()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource, release).
					WithInterceptorFuncs(interceptor.Funcs{
						Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
							return assert.AnError
						},
					}).Build()

				wrappedErr := fmt.Errorf("failed to delete HelmRelease %q: %w", testDoguName, assert.AnError)
				recorder := NewMockEventRecorder(t)
				recorder.EXPECT().Event(doguResource, v1.EventTypeWarning, stepsv3.ReasonDeletionFailed, wrappedErr.Error())
				return c, recorder, doguResource
			},
			want: stepsv3.RequeueWithError(fmt.Errorf("failed to delete HelmRelease %q: %w", testDoguName, assert.AnError), stepsv3.ReasonDeletionFailed),
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
			sut := NewDeleteHelmReleaseStep(c, recorder)

			if got := sut.Run(testCtx, doguResource); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() = %v, want %v", got, tt.want)
			}

			if tt.assertFn != nil {
				tt.assertFn(t, c)
			}
		})
	}
}

func TestNewDeleteHelmReleaseStep(t *testing.T) {
	clientMock := NewMockK8sClient(t)
	recorderMock := NewMockEventRecorder(t)

	sut := NewDeleteHelmReleaseStep(clientMock, recorderMock)

	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.client)
	assert.Equal(t, recorderMock, sut.eventRecorder)
}
