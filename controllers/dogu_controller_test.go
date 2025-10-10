package controllers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

func TestNewDoguReconciler(t *testing.T) {
	// given
	managerMock := newMockCtrlManager(t)
	managerMock.EXPECT().GetControllerOptions().Return(config.Controller{})
	managerMock.EXPECT().GetScheme().Return(getTestScheme())
	managerMock.EXPECT().GetLogger().Return(logr.Logger{})
	managerMock.EXPECT().Add(mock.Anything).Return(nil)
	managerMock.EXPECT().GetCache().Return(nil)
	managerMock.EXPECT().GetRESTMapper().Return(nil)

	// when
	reconciler, err := NewDoguReconciler(nil, nil, nil, nil, nil, nil, nil, managerMock)

	// then
	assert.NoError(t, err)
	assert.NotNil(t, reconciler)
}

func TestDoguReconciler_Reconcile(t *testing.T) {
	type fields struct {
		clientFn            func(t *testing.T) client.Client
		doguChangeHandlerFn func(t *testing.T) DoguUsecase
		doguDeleteHandlerFn func(t *testing.T) DoguUsecase
		doguInterfaceFn     func(t *testing.T) doguInterface
		requeueHandlerFn    func(t *testing.T) RequeueHandler
		eventRecorderFn     func(t *testing.T) eventRecorder
	}
	tests := []struct {
		name    string
		fields  fields
		req     controllerruntime.Request
		want    controllerruntime.Result
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get dogu resource",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					mck := NewMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{}, &v2.Dogu{}).Return(assert.AnError)
					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, &v2.Dogu{}, assert.AnError, time.Duration(0)).Return(controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{},
			want:    controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
		{
			name: "should stop if dogu resource not found",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					scheme := runtime.NewScheme()
					err := v2.AddToScheme(scheme)
					require.NoError(t, err)
					mck := fake.NewClientBuilder().
						WithScheme(scheme).
						Build()

					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, &v2.Dogu{}, nil, time.Duration(0)).Return(controllerruntime.Result{Requeue: false, RequeueAfter: 0}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{},
			want:    controllerruntime.Result{Requeue: false, RequeueAfter: 0},
			wantErr: assert.NoError,
		},
		{
			name: "should fail to update dogu resource",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					scheme := runtime.NewScheme()
					err := v2.AddToScheme(scheme)
					require.NoError(t, err)
					doguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck := fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(doguResource).
						Build()

					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					mck := NewMockDoguUsecase(t)
					mck.EXPECT().HandleUntilApplied(testCtx, mock.Anything).Return(0, false, nil)
					return mck
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.AnythingOfType("*v2.Dogu"), v3.EventTypeNormal, ReasonReconcileStarted, "reconciliation started")
					return mck
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, mock.AnythingOfType("*v2.Dogu"), errors.Join(assert.AnError), time.Duration(0)).Return(controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
		{
			name: "should succeed to update dogu resource on error",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					scheme := runtime.NewScheme()
					err := v2.AddToScheme(scheme)
					require.NoError(t, err)
					doguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck := fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(doguResource).
						Build()

					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					mck := NewMockDoguUsecase(t)
					mck.EXPECT().HandleUntilApplied(testCtx, mock.Anything).Return(0, false, assert.AnError)
					return mck
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}, nil)
					return mck
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.AnythingOfType("*v2.Dogu"), v3.EventTypeNormal, ReasonReconcileStarted, "reconciliation started")
					return mck
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, mock.AnythingOfType("*v2.Dogu"), errors.Join(assert.AnError), time.Duration(0)).Return(controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{Requeue: true, RequeueAfter: 5 * time.Second},
			wantErr: assert.NoError,
		},
		{
			name: "should succeed to update dogu resource on abort",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					scheme := runtime.NewScheme()
					err := v2.AddToScheme(scheme)
					require.NoError(t, err)
					doguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck := fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(doguResource).
						Build()

					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					mck := NewMockDoguUsecase(t)
					mck.EXPECT().HandleUntilApplied(testCtx, mock.Anything).Return(0, true, nil)
					return mck
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}, nil)
					return mck
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.AnythingOfType("*v2.Dogu"), v3.EventTypeNormal, ReasonReconcileStarted, "reconciliation started")
					return mck
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, mock.AnythingOfType("*v2.Dogu"), nil, time.Duration(0)).Return(controllerruntime.Result{Requeue: false, RequeueAfter: 0}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{RequeueAfter: 0},
			wantErr: assert.NoError,
		},
		{
			name: "should succeed to update dogu resource on continue",
			fields: fields{
				clientFn: func(t *testing.T) client.Client {
					scheme := runtime.NewScheme()
					err := v2.AddToScheme(scheme)
					require.NoError(t, err)
					doguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck := fake.NewClientBuilder().
						WithScheme(scheme).
						WithObjects(doguResource).
						Build()

					return mck
				},
				doguChangeHandlerFn: func(t *testing.T) DoguUsecase {
					mck := NewMockDoguUsecase(t)
					mck.EXPECT().HandleUntilApplied(testCtx, mock.Anything).Return(0, false, nil)
					return mck
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, mock.Anything, mock.Anything, v1.UpdateOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}, nil)
					return mck
				},
				eventRecorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.AnythingOfType("*v2.Dogu"), v3.EventTypeNormal, ReasonReconcileStarted, "reconciliation started")
					return mck
				},
				requeueHandlerFn: func(t *testing.T) RequeueHandler {
					mck := NewMockRequeueHandler(t)
					mck.EXPECT().Handle(testCtx, mock.AnythingOfType("*v2.Dogu"), nil, time.Duration(0)).Return(controllerruntime.Result{Requeue: false, RequeueAfter: 0}, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{RequeueAfter: 0},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &DoguReconciler{
				client:            tt.fields.clientFn(t),
				doguChangeHandler: tt.fields.doguChangeHandlerFn(t),
				doguDeleteHandler: tt.fields.doguDeleteHandlerFn(t),
				doguInterface:     tt.fields.doguInterfaceFn(t),
				requeueHandler:    tt.fields.requeueHandlerFn(t),
				eventRecorder:     tt.fields.eventRecorderFn(t),
			}
			got, err := r.Reconcile(testCtx, tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v, %v)", testCtx, tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Reconcile(%v, %v)", testCtx, tt.req)
		})
	}
}
