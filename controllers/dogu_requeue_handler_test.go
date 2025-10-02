package controllers

import (
	"fmt"
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v2 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func TestNewDoguRequeueHandler(t *testing.T) {
	t.Run("should fail to create DoguRequeueHandler", func(t *testing.T) {
		// given
		conf := &config.OperatorConfig{}
		doguInterfaceMock := newMockDoguInterface(t)
		eventRecorderMock := newMockEventRecorder(t)

		// when
		handler, err := NewDoguRequeueHandler(doguInterfaceMock, eventRecorderMock, conf)

		// then
		assert.Error(t, err)
		assert.Empty(t, handler)
	})
}

func Test_doguRequeueHandler_Handle(t *testing.T) {
	type fields struct {
		nonCacheClientFn func(t *testing.T) kubernetes.Interface
		recorderFn       func(t *testing.T) record.EventRecorder
		doguInterfaceFn  func(t *testing.T) client.DoguInterface
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
			name: "should stop if no error or requeueTime",
			fields: fields{
				nonCacheClientFn: func(t *testing.T) kubernetes.Interface {
					return NewMockClientSet(t)
				},
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Event(mock.AnythingOfType("*v2.Dogu"), v2.EventTypeNormal, ReasonReconcileOK, "resource synced")
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						Status: doguv2.DoguStatus{
							RequeueTime:  0,
							RequeuePhase: "",
						},
					}, v1.UpdateOptions{}).Return(&doguv2.Dogu{}, nil)
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
			name: "should fail to set requeueTime in dogu cr",
			fields: fields{
				nonCacheClientFn: func(t *testing.T) kubernetes.Interface {
					return NewMockClientSet(t)
				},
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						Status: doguv2.DoguStatus{
							RequeueTime:  5 * time.Second,
							RequeuePhase: "",
						},
					}, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{},
				err:          fmt.Errorf(""),
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime},
			wantErr: assert.Error,
		},
		{
			name: "should succeed handler",
			fields: fields{
				nonCacheClientFn: func(t *testing.T) kubernetes.Interface {
					eventMock := newMockEventInterface(t)
					eventMock.EXPECT().List(testCtx, v1.ListOptions{
						FieldSelector: fmt.Sprintf("reason=%s,involvedObject.name=%s", RequeueEventReason, ""),
					}).Return(&v2.EventList{Items: []v2.Event{{ObjectMeta: v1.ObjectMeta{Name: "event"}}}}, nil)
					eventMock.EXPECT().Delete(testCtx, "event", v1.DeleteOptions{}).Return(nil)
					coreV1Mock := newMockCoreV1Interface(t)
					coreV1Mock.EXPECT().Events("ecosystem").Return(eventMock)
					mck := NewMockClientSet(t)
					mck.EXPECT().CoreV1().Return(coreV1Mock)
					return mck
				},
				recorderFn: func(t *testing.T) record.EventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Eventf(&doguv2.Dogu{}, v2.EventTypeNormal, RequeueEventReason, "Trying again in %s.", requeueTime.String())
					return mck
				},
				doguInterfaceFn: func(t *testing.T) client.DoguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &doguv2.Dogu{
						Status: doguv2.DoguStatus{
							RequeueTime:  5 * time.Second,
							RequeuePhase: "",
						},
					}, v1.UpdateOptions{}).Return(&doguv2.Dogu{}, nil)
					return mck
				},
			},
			args: args{
				doguResource: &doguv2.Dogu{},
				err:          fmt.Errorf(""),
				reqTime:      time.Duration(0),
			},
			want:    controllerruntime.Result{Requeue: true, RequeueAfter: requeueTime},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &doguRequeueHandler{
				nonCacheClient: tt.fields.nonCacheClientFn(t),
				namespace:      "ecosystem",
				recorder:       tt.fields.recorderFn(t),
				doguInterface:  tt.fields.doguInterfaceFn(t),
			}
			got, err := d.Handle(testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)
			if !tt.wantErr(t, err, fmt.Sprintf("Handle(%v, %v, %v, %v)", testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Handle(%v, %v, %v, %v)", testCtx, tt.args.doguResource, tt.args.err, tt.args.reqTime)
		})
	}
}
