package controllers

import (
	context "context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const requeueTime = time.Second * 5

func TestNewDoguRequeueHandler(t *testing.T) {
	t.Run("should fail to create DoguRequeueHandler", func(t *testing.T) {
		// given
		conf := &config.OperatorConfig{Namespace: testNamespace}
		doguInterfaceMock := newMockDoguInterface(t)
		eventRecorderMock := newMockEventRecorder(t)

		// when
		handler := NewDoguRequeueHandler(doguInterfaceMock, eventRecorderMock, conf)

		// then
		assert.Same(t, doguInterfaceMock, handler.doguInterface)
		assert.Same(t, eventRecorderMock, handler.recorder)
		assert.Equal(t, testNamespace, handler.namespace)
	})
}

func Test_doguRequeueHandler_Handle(t *testing.T) {
	type fields struct {
		recorderFn      func(t *testing.T) record.EventRecorder
		doguInterfaceFn func(t *testing.T) client.DoguInterface
	}
	type args struct {
		doguResource *doguv2.Dogu
		err          error
		reqTime      time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    controllerruntime.Result
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should reconcile on error",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					err := errors.New("Reconciliation failed")

					mck.EXPECT().Eventf(
						&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						v2.EventTypeWarning,
						ReasonReconcileFail,
						"Trying again in %s because of: %s", requeueTime.String(), err.Error()).Return()
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					getDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					updateDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck.EXPECT().Get(testCtx, testDoguName, v1.GetOptions{}).Return(getDogu, nil)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, getDogu, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *doguv2.Dogu, modifyStatusFn func(doguv2.DoguStatus) doguv2.DoguStatus, opts v1.UpdateOptions) {
						status := modifyStatusFn(dogu.Status)
						assert.Equal(t, requeueTime, status.RequeueTime)
					}).Return(updateDogu, nil)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
				err:          errors.New("Reconciliation failed"),
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
		{
			name: "should reconcile on error updating dogu resource",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					err := errors.New("Reconciliation failed")

					mck.EXPECT().Eventf(
						&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						v2.EventTypeWarning,
						ReasonReconcileFail,
						"Trying again in %s because of: %s", requeueTime.String(), err.Error()).Return()
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					getDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					mck.EXPECT().Get(testCtx, testDoguName, v1.GetOptions{}).Return(getDogu, nil)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, getDogu, mock.Anything, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
				err:          errors.New("Reconciliation failed"),
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
		{
			name: "should reconcile on error getting dogu resource",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					err := errors.New("Reconciliation failed")

					mck.EXPECT().Eventf(
						&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						v2.EventTypeWarning,
						ReasonReconcileFail,
						"Trying again in %s because of: %s", requeueTime.String(), err.Error()).Return()
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().Get(testCtx, testDoguName, v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
				err:          errors.New("Reconciliation failed"),
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
		{
			name: "should reconcile on requeue time",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					reqTime := 15 * time.Second
					mck.EXPECT().Eventf(
						&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						v2.EventTypeNormal,
						RequeueEventReason,
						"Trying again in %s.", reqTime.String(),
					).Return()
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					getDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					updateDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					updateDogu.Status.RequeueTime = 15 * time.Second
					mck.EXPECT().Get(testCtx, testDoguName, v1.GetOptions{}).Return(getDogu, nil)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, getDogu, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *doguv2.Dogu, modifyStatusFn func(doguv2.DoguStatus) doguv2.DoguStatus, opts v1.UpdateOptions) {
						status := modifyStatusFn(dogu.Status)
						assert.Equal(t, 15*time.Second, status.RequeueTime)
					}).Return(updateDogu, nil)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
				err:          nil,
				reqTime:      15 * time.Second,
			},
			want:    controllerruntime.Result{RequeueAfter: 15 * time.Second},
			wantErr: assert.NoError,
		},
		{
			name: "should not reconcile when no error or no requeue time",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(
						&doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						v2.EventTypeNormal,
						ReasonReconcileOK,
						"resource synced").Return()
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					getDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					updateDogu := &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}}
					updateDogu.Status.RequeueTime = 0
					mck.EXPECT().Get(testCtx, testDoguName, v1.GetOptions{}).Return(getDogu, nil)
					mck.EXPECT().UpdateStatusWithRetry(testCtx, getDogu, mock.Anything, v1.UpdateOptions{}).Run(func(ctx context.Context, dogu *doguv2.Dogu, modifyStatusFn func(doguv2.DoguStatus) doguv2.DoguStatus, opts v1.UpdateOptions) {
						status := modifyStatusFn(dogu.Status)
						assert.Equal(t, time.Duration(0), status.RequeueTime)
					}).Return(updateDogu, nil)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
				err:          nil,
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.NoError,
		},
		{
			name: "should not reconcile when dogu is empty",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{},
				err:          nil,
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.NoError,
		},
		{
			name: "should not reconcile when deletion timestamp is set",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{ObjectMeta: v1.ObjectMeta{Name: testDoguName, DeletionTimestamp: &v1.Time{Time: time.Now()}}},
				err:          errors.New(""),
				reqTime:      time.Duration(2),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &doguRequeueHandler{
				namespace:     "ecosystem",
				recorder:      tt.fields.recorderFn(t),
				doguInterface: tt.fields.doguInterfaceFn(t),
				requeueTime:   time.Second * 5,
			}
			got, err := d.Handle(testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)
			if !tt.wantErr(t, err, fmt.Sprintf("Handle(%v, %v, %v, %v)", testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Handle(%v, %v, %v, %v)", testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)
		})
	}
}
