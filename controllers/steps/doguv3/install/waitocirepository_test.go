package install

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/fluxcd/pkg/apis/meta"
	flux "github.com/fluxcd/source-controller/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var (
	readyRepository = &flux.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDoguName,
		},
		Status: flux.OCIRepositoryStatus{
			Conditions: []metav1.Condition{
				{
					Type:   meta.ReadyCondition,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}
	conflictErr = apierrors.NewConflict(v3beta1.GroupVersion.WithResource("dogus").GroupResource(), testDoguName, assert.AnError)
)

func TestWaitForOCIRepositoryReadyStep_Run(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu)
		want     doguv3.StepResult
		assertFn func(t *testing.T, client K8sClient)
	}{
		{
			name: "should continue and update status on dogu resource if ocirepository is ready",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(readyRepository, doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()

				recorderMock := NewMockEventRecorder(t)
				recorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, v3beta1.ConditionChartAvailable, successMessage)

				return c, recorderMock, doguResource
			},
			want: testStepResultContinue,
			assertFn: func(t *testing.T, client K8sClient) {
				dogu := &v3beta1.Dogu{}
				err := client.Get(testCtx, testNamespacedName, dogu)

				require.NoError(t, err)
				conditions := dogu.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonSucceeded, conditions[0].Reason)
				assert.Equal(t, successMessage, conditions[0].Message)
			},
		},
		{
			name: "should continue and not writing dogu status if repository is ready and dogu resource contains success status condition",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Status.Conditions = []metav1.Condition{
					{
						Type:    v3beta1.ConditionChartAvailable,
						Status:  metav1.ConditionTrue,
						Reason:  v3beta1.ReasonSucceeded,
						Message: successMessage,
					},
				}
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(readyRepository).Build()

				return c, nil, doguResource
			},
			want: testStepResultContinue,
		},
		{
			name: "should requeue without error if the dogu resource is not found",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				c := fake.NewClientBuilder().WithScheme(testScheme).Build()
				return c, nil, testDoguResource.DeepCopy()
			},
			want: doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, "ocirepositories.source.toolkit.fluxcd.io \"nexus\" not found"),
		},
		{
			name: "should requeue with error on non 404 error getting dogu resource",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				c := fake.NewClientBuilder().WithScheme(testScheme).WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						return assert.AnError
					},
				}).Build()
				return c, nil, testDoguResource.DeepCopy()
			},
			want: doguv3.RequeueWithError(fmt.Errorf("failed to get OCIRepository: %w", assert.AnError), v3beta1.ReasonInstalling),
		},
		{
			name: "should process update chartavailable status if oci repository is not ready",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				notReadyRepository := &flux.OCIRepository{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testNamespace,
						Name:      testDoguName,
					},
				}
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(notReadyRepository, doguResource).Build()
				return c, nil, doguResource
			},
			want: doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, ""),
		},
		{
			name: "should requeue without error if repo is ready but conflict error on writing dogu status condition",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(readyRepository, doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							if subResourceName == "status" {
								return conflictErr
							}
							return c.SubResource(subResourceName).Update(ctx, obj, opts...)
						},
					}).Build()
				return c, nil, doguResource
			},
			want: doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, conflictErr.Error()),
		},
		{
			name: "should requeue with error if repo is ready but non conflict error on writing dogu status condition",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(readyRepository, doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							if subResourceName == "status" {
								return assert.AnError
							}
							return c.SubResource(subResourceName).Update(ctx, obj, opts...)
						},
					}).Build()
				return c, nil, doguResource
			},
			want: doguv3.RequeueWithError(fmt.Errorf("%s: %w", "failed to update oci success condition", assert.AnError), v3beta1.DoguStatusInstalling),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient, eventRecorder, doguResource := tt.setup(t)
			wor := &WaitForOCIRepositoryReadyStep{k8sClient: k8sClient, eventRecorder: eventRecorder}

			assert.Equalf(t, tt.want, wor.Run(testCtx, doguResource), "Run(%v, %v)", testCtx, doguResource)

			if tt.assertFn != nil {
				tt.assertFn(t, wor.k8sClient)
			}
		})
	}
}

func TestWaitForOCIRepositoryReadyStep_updateChartUnavailableStatus(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu)
		repo     *flux.OCIRepository
		want     doguv3.StepResult
		assertFn func(t *testing.T, doguResource *v3beta1.Dogu)
	}{
		{
			name: "should requeue after default duration when reason could not be determined e.g. no repo conditions",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				return nil, nil, testDoguResource.DeepCopy()
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
			},
			want: doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, ""),
		},
		{
			name: "should abort with download failed reason using the ready condition message as fallback when no fetch failed condition is present",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{
							Type:    meta.ReadyCondition,
							Status:  metav1.ConditionFalse,
							Message: "other error",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonDownloadFailed, "other error"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonDownloadFailed, conditions[0].Reason)
				assert.Equal(t, "other error", conditions[0].Message)
			},
		},
		{
			name: "should abort with download failed reason when fetch failed due to an authentication error",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse},
						{
							Type:    flux.FetchFailedCondition,
							Status:  metav1.ConditionTrue,
							Reason:  flux.AuthenticationFailedReason,
							Message: "invalid credentials",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonDownloadFailed, "invalid credentials"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonDownloadFailed, conditions[0].Reason)
				assert.Equal(t, "invalid credentials", conditions[0].Message)
			},
		},
		{
			name: "should abort with unauthorized reason when oci pull fails due to unauthorized access",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse},
						{
							Type:    flux.FetchFailedCondition,
							Status:  metav1.ConditionTrue,
							Reason:  flux.OCIPullFailedReason,
							Message: "failed to pull chart: unauthorized to access repository",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonUnauthorized, "failed to pull chart: unauthorized to access repository"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonUnauthorized, conditions[0].Reason)
				assert.Equal(t, "failed to pull chart: unauthorized to access repository", conditions[0].Message)
			},
		},
		{
			name: "should abort with chart not found reason when oci pull fails because the chart is missing",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse},
						{
							Type:    flux.FetchFailedCondition,
							Status:  metav1.ConditionTrue,
							Reason:  flux.OCIPullFailedReason,
							Message: "manifest not found in registry",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonChartNotFound, "manifest not found in registry"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonChartNotFound, conditions[0].Reason)
				assert.Equal(t, "manifest not found in registry", conditions[0].Message)
			},
		},
		{
			name: "should abort with download failed reason when oci pull fails for a reason other than unauthorized or not found",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse},
						{
							Type:    flux.FetchFailedCondition,
							Status:  metav1.ConditionTrue,
							Reason:  flux.OCIPullFailedReason,
							Message: "unexpected registry error",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonDownloadFailed, "unexpected registry error"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonDownloadFailed, conditions[0].Reason)
				assert.Equal(t, "unexpected registry error", conditions[0].Message)
			},
		},
		{
			name: "should abort with download failed reason when fetch failed for an unknown reason",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{Type: meta.ReadyCondition, Status: metav1.ConditionFalse},
						{
							Type:    flux.FetchFailedCondition,
							Status:  metav1.ConditionTrue,
							Reason:  "SomeOtherReason",
							Message: "something else failed",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonDownloadFailed, "something else failed"),
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionChartAvailable, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonDownloadFailed, conditions[0].Reason)
				assert.Equal(t, "something else failed", conditions[0].Message)
			},
		},
		{
			name: "should abort without persisting status when failure condition did not change",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				doguResource.Status.Conditions = []metav1.Condition{
					{
						Type:    v3beta1.ConditionChartAvailable,
						Status:  metav1.ConditionFalse,
						Reason:  v3beta1.ReasonDownloadFailed,
						Message: "chart download timed out",
					},
				}
				c := fake.NewClientBuilder().WithScheme(testScheme).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{
							Type:    meta.ReadyCondition,
							Status:  metav1.ConditionFalse,
							Message: "chart download timed out",
						},
					},
				},
			},
			want: doguv3.Abort(v3beta1.ReasonDownloadFailed, "chart download timed out"),
		},
		{
			name: "should requeue after default duration when persisting failure status conflicts",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							if subResourceName == "status" {
								return conflictErr
							}
							return c.SubResource(subResourceName).Update(ctx, obj, opts...)
						},
					}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{
							Type:    meta.ReadyCondition,
							Status:  metav1.ConditionFalse,
							Message: "chart download timed out",
						},
					},
				},
			},
			want: doguv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, conflictErr.Error()),
		},
		{
			name: "should requeue with error when persisting failure status fails unexpectedly",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu) {
				doguResource := testDoguResource.DeepCopy()
				c := fake.NewClientBuilder().
					WithScheme(testScheme).
					WithObjects(doguResource).
					WithStatusSubresource(&v3beta1.Dogu{}).
					WithInterceptorFuncs(interceptor.Funcs{
						SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
							if subResourceName == "status" {
								return assert.AnError
							}
							return c.SubResource(subResourceName).Update(ctx, obj, opts...)
						},
					}).Build()
				return c, nil, doguResource
			},
			repo: &flux.OCIRepository{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testNamespace,
					Name:      testDoguName,
				},
				Status: flux.OCIRepositoryStatus{
					Conditions: []metav1.Condition{
						{
							Type:    meta.ReadyCondition,
							Status:  metav1.ConditionFalse,
							Message: "chart download timed out",
						},
					},
				},
			},
			want: doguv3.RequeueWithError(fmt.Errorf("%s: %w", "failed to update oci failure condition", assert.AnError), v3beta1.DoguStatusInstalling),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient, eventRecorder, doguResource := tt.setup(t)
			wor := &WaitForOCIRepositoryReadyStep{k8sClient: k8sClient, eventRecorder: eventRecorder}

			assert.Equalf(t, tt.want, wor.updateChartUnavailableStatus(testCtx, doguResource, tt.repo), "updateChartUnavailableStatus(%v, %v, %v)", testCtx, doguResource, tt.repo)

			if tt.assertFn != nil {
				tt.assertFn(t, doguResource)
			}
		})
	}
}

func TestNewWaitForOCIRepositoryReadyStep(t *testing.T) {
	// given
	clientMock := NewMockK8sClient(t)
	recorderMock := NewMockEventRecorder(t)

	// when
	sut := NewWaitForOCIRepositoryReadyStep(clientMock, recorderMock)

	// then
	assert.NotNil(t, sut)
	assert.Equal(t, clientMock, sut.k8sClient)
	assert.Equal(t, recorderMock, sut.eventRecorder)
}
