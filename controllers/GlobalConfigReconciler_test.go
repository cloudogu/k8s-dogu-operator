package controllers

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const testDoguName = "test"
const testNamespace = "ecosystem"

func TestNewGlobalConfigReconciler(t *testing.T) {
	// given
	doguInterfaceMock := newMockDoguInterface(t)
	managerMock := newMockCtrlManager(t)
	managerMock.EXPECT().GetControllerOptions().Return(config.Controller{})
	managerMock.EXPECT().GetScheme().Return(getTestScheme())
	managerMock.EXPECT().GetLogger().Return(logr.Logger{})
	managerMock.EXPECT().Add(mock.Anything).Return(nil)
	managerMock.EXPECT().GetCache().Return(nil)

	// when
	reconciler, err := NewGlobalConfigReconciler(doguInterfaceMock, nil, managerMock)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, reconciler)
}

func TestGlobalConfigReconciler_Reconcile(t *testing.T) {
	type fields struct {
		doguInterfaceFn func(t *testing.T) doguInterface
		doguEvents      chan<- event.TypedGenericEvent[*v2.Dogu]
	}
	tests := []struct {
		name    string
		fields  fields
		req     controllerruntime.Request
		want    controllerruntime.Result
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to list dogus",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().List(testCtx, v1.ListOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &GlobalConfigReconciler{
				doguInterface: tt.fields.doguInterfaceFn(t),
				doguEvents:    tt.fields.doguEvents,
			}
			got, err := r.Reconcile(testCtx, tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v, %v)", testCtx, tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Reconcile(%v, %v)", testCtx, tt.req)
		})
	}
}
