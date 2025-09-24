package controllers

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	reconciler, err := NewDoguReconciler(nil, nil, nil, nil, nil, managerMock)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, reconciler)
}

func TestDoguReconciler_Reconcile(t *testing.T) {
	type fields struct {
		clientFn            func(t *testing.T) client.Client
		doguChangeHandlerFn func(t *testing.T) DoguUsecase
		doguDeleteHandlerFn func(t *testing.T) DoguUsecase
		doguInterfaceFn     func(t *testing.T) doguInterface
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
			},
			req:     controllerruntime.Request{},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
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
					mck.EXPECT().HandleUntilApplied(testCtx, mock.Anything).Return(1, false, nil)
					return mck
				},
				doguDeleteHandlerFn: func(t *testing.T) DoguUsecase {
					return NewMockDoguUsecase(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, mock.Anything, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{RequeueAfter: 1},
			wantErr: assert.Error,
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
					mck.EXPECT().UpdateStatus(testCtx, mock.Anything, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
			},
			req:     controllerruntime.Request{NamespacedName: types.NamespacedName{Name: testDoguName}},
			want:    controllerruntime.Result{RequeueAfter: 0},
			wantErr: assert.Error,
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
					mck.EXPECT().UpdateStatus(testCtx, mock.Anything, v1.UpdateOptions{}).Return(nil, nil)
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
					mck.EXPECT().UpdateStatus(testCtx, mock.Anything, v1.UpdateOptions{}).Return(nil, nil)
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
			}
			got, err := r.Reconcile(testCtx, tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v, %v)", testCtx, tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Reconcile(%v, %v)", testCtx, tt.req)
		})
	}
}
