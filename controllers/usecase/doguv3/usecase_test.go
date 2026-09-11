package doguv3

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace     = "ecosystem"
	testVersion       = "1.0.0"
	testDoguName      = "nexus"
	testDoguNamespace = "testing"
)

var (
	testCtx          = context.Background()
	testDoguResource = &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testDoguName,
			Namespace: testNamespace,
		},
		Spec: v3beta1.DoguSpec{
			Name:          testDoguName,
			DoguNamespace: testDoguNamespace,
			Version:       testVersion,
		},
		Status: v3beta1.DoguStatus{
			Conditions: []metav1.Condition{},
		},
	}
	testScheme  = runtime.NewScheme()
	_           = v3beta1.AddToScheme(testScheme)
	conflictErr = apierrors.NewConflict(v3beta1.GroupVersion.WithResource("dogus").GroupResource(), testDoguName, assert.AnError)
)

func TestNewDoguInstallOrChangeUseCase(t *testing.T) {
	clientMock := NewMockK8sClient(t)
	recorderMock := NewMockEventRecorder(t)
	ensureOCIStep := &install.EnsureOCIRepositoryStep{}
	waitOCIStep := &install.WaitForOCIRepositoryReadyStep{}

	got := NewDoguInstallOrChangeUseCase(
		ensureOCIStep,
		waitOCIStep,
		clientMock,
		recorderMock,
	)

	wantTypes := []string{
		"*install.EnsureOCIRepositoryStep",
		"*install.WaitForOCIRepositoryReadyStep",
	}

	assert.NotNil(t, got)
	assert.Equal(t, clientMock, got.k8sClient)
	assert.Equal(t, recorderMock, got.eventRecorder)
	require.True(t,
		slices.Equal(typesOf(got.steps), wantTypes),
		"order mismatch: got=%v want=%v",
		typesOf(got.steps), wantTypes,
	)
}

func TestNewDoguDeleteUseCase(t *testing.T) {
	clientMock := NewMockK8sClient(t)

	got := NewDoguDeleteUseCase(clientMock)

	wantTypes := []string{}

	assert.NotNil(t, got)
	assert.Equal(t, clientMock, got.k8sClient)
	require.True(t,
		slices.Equal(typesOf(got.steps), wantTypes),
		"order mismatch: got=%v want=%v",
		typesOf(got.steps), wantTypes,
	)
}

func typesOf[T any](xs []T) []string {
	out := make([]string, len(xs))
	for i, v := range xs {
		out[i] = reflect.TypeOf(v).String()
	}
	return out
}

func TestDoguUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step)
		want     time.Duration
		wantErr  assert.ErrorAssertionFunc
		assertFn func(t *testing.T, doguResource *v3beta1.Dogu)
	}{
		{
			name: "should set ready condition true on successful steps",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				successStepResult := doguv3.StepResult{Continue: true}
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(successStepResult)
				stepMock2.EXPECT().Run(testCtx, doguResource).Return(successStepResult)

				eventRecorderMock := NewMockEventRecorder(t)
				eventRecorderMock.EXPECT().Event(doguResource, v1.EventTypeNormal, ReasonReconcileSuccess, "Dogu reconciliation completed successfully")

				clientMock := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(&v3beta1.Dogu{}).Build()

				return clientMock, eventRecorderMock, doguResource, []Step{stepMock1, stepMock2}
			},
			want:    0,
			wantErr: assert.NoError,
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionReady, conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonSucceeded, conditions[0].Reason)
				assert.Equal(t, "Dogu reconciliation completed successfully", conditions[0].Message)
			},
		},
		{
			name: "should requeue without error on conflict error writing status",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				successStepResult := doguv3.StepResult{Continue: true}
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(successStepResult)
				stepMock2.EXPECT().Run(testCtx, doguResource).Return(successStepResult)

				clientMock := fake.NewClientBuilder().
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

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want:    5 * time.Second,
			wantErr: assert.NoError,
		},
		{
			name: "should requeue with error on non conflict error writing status",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				successStepResult := doguv3.StepResult{Continue: true}
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(successStepResult)
				stepMock2.EXPECT().Run(testCtx, doguResource).Return(successStepResult)

				clientMock := fake.NewClientBuilder().
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

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i...)
			},
		},
		{
			name: "should set ready condition false and send event when a step aborts",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				abortResult := doguv3.Abort(v3beta1.ReasonDownloadFailed, "chart not found")
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(abortResult)

				eventRecorderMock := NewMockEventRecorder(t)
				eventRecorderMock.EXPECT().Event(doguResource, v1.EventTypeWarning, v3beta1.ReasonDownloadFailed, "chart not found")

				clientMock := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(&v3beta1.Dogu{}).Build()

				return clientMock, eventRecorderMock, doguResource, []Step{stepMock1, stepMock2}
			},
			want:    0,
			wantErr: assert.NoError,
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionReady, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonDownloadFailed, conditions[0].Reason)
				assert.Equal(t, "chart not found", conditions[0].Message)
			},
		},
		{
			name: "should set ready condition false and requeue without event when a step requests a requeue",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				requeueResult := doguv3.RequeueAfter(5*time.Second, v3beta1.ReasonInstalling, "waiting for OCIRepository")
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(requeueResult)

				clientMock := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(&v3beta1.Dogu{}).Build()

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want:    5 * time.Second,
			wantErr: assert.NoError,
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionReady, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonInstalling, conditions[0].Reason)
				assert.Equal(t, "waiting for OCIRepository", conditions[0].Message)
			},
		},
		{
			name: "should set ready condition false and requeue with error when a step errors",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				requeueWithErrResult := doguv3.RequeueWithError(assert.AnError, v3beta1.ReasonInstalling)
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(requeueWithErrResult)

				clientMock := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(&v3beta1.Dogu{}).Build()

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i...)
			},
			assertFn: func(t *testing.T, doguResource *v3beta1.Dogu) {
				conditions := doguResource.Status.Conditions
				assert.Len(t, conditions, 1)
				assert.Equal(t, v3beta1.ConditionReady, conditions[0].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, v3beta1.ReasonInstalling, conditions[0].Reason)
				assert.Equal(t, assert.AnError.Error(), conditions[0].Message)
			},
		},
		{
			name: "should requeue without error when persisting the not-ready status conflicts",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				abortResult := doguv3.Abort(v3beta1.ReasonDownloadFailed, "chart not found")
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(abortResult)

				clientMock := fake.NewClientBuilder().
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

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want:    5 * time.Second,
			wantErr: assert.NoError,
		},
		{
			name: "should combine step error and persist error when persisting the not-ready status fails with a non-conflict error",
			setup: func(t *testing.T) (K8sClient, EventRecorder, *v3beta1.Dogu, []Step) {
				requeueWithErrResult := doguv3.RequeueWithError(assert.AnError, v3beta1.ReasonInstalling)
				doguResource := testDoguResource.DeepCopy()
				stepMock1 := NewMockStep(t)
				stepMock2 := NewMockStep(t)
				stepMock1.EXPECT().Run(testCtx, doguResource).Return(requeueWithErrResult)

				clientMock := fake.NewClientBuilder().
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

				return clientMock, nil, doguResource, []Step{stepMock1, stepMock2}
			},
			want: 0,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i...)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient, eventRecorder, doguResource, steps := tt.setup(t)
			duc := &DoguUseCase{
				steps:         steps,
				k8sClient:     k8sClient,
				eventRecorder: eventRecorder,
			}
			got, err := duc.HandleUntilApplied(testCtx, doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("HandleUntilApplied(%v, %v)", testCtx, doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "HandleUntilApplied(%v, %v)", testCtx, doguResource)

			if tt.assertFn != nil {
				tt.assertFn(t, doguResource)
			}
		})
	}
}

func Test_deriveReasonAndMessage(t *testing.T) {
	type args struct {
		result doguv3.StepResult
	}
	tests := []struct {
		name        string
		args        args
		wantReason  string
		wantMessage string
	}{
		{
			name: "if no reason and message should return StepFailed reason and error message on error",
			args: args{
				result: doguv3.StepResult{Err: assert.AnError},
			},
			wantReason:  "StepFailed",
			wantMessage: assert.AnError.Error(),
		},
		{
			name: "if no reason and message should return Installing reason and wait message on no error but requeue after",
			args: args{
				result: doguv3.StepResult{RequeueAfter: 5 * time.Second},
			},
			wantReason:  v3beta1.ReasonInstalling,
			wantMessage: "Waiting for background operation to complete",
		},
		{
			name: "should return aborted reason on no error and no requeue after",
			args: args{
				result: doguv3.StepResult{},
			},
			wantReason:  "ReconciliationAborted",
			wantMessage: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, message := deriveReasonAndMessage(tt.args.result)
			assert.Equalf(t, tt.wantReason, reason, "deriveReasonAndMessage(%v)", tt.args.result)
			assert.Equalf(t, tt.wantMessage, message, "deriveReasonAndMessage(%v)", tt.args.result)
		})
	}
}
