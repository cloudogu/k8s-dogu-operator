package controllers

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const testDoguName = "test"
const testNamespace = "ecosystem"

func TestGlobalConfigReconcilerV3Dispatch(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("V3 enabled=%t", enabled), func(t *testing.T) {
			legacy := v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "legacy", Namespace: testNamespace}}
			modern := v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "modern", Namespace: testNamespace,
					Annotations: map[string]string{"k8s.cloudogu.com/v3beta1-doguApiVersion": "v3"}},
				Spec: v2.DoguSpec{Name: "official/modern"},
			}
			dogus := newMockDoguInterface(t)
			dogus.EXPECT().List(testCtx, v1.ListOptions{}).Return(&v2.DoguList{Items: []v2.Dogu{legacy, modern}}, nil)
			events := make(chan event.TypedGenericEvent[*v2.Dogu], 2)
			r := &GlobalConfigReconciler{doguInterface: dogus, doguEvents: events}
			_, err := r.Reconcile(testCtx, controllerruntime.Request{})
			require.NoError(t, err)
			require.Len(t, events, 2)
			assert.Equal(t, "legacy", (<-events).Object.Name, "V2 fan-out remains intact")
			modernEvent := <-events
			assert.Equal(t, "modern", modernEvent.Object.Name)
			handler := NewMockDoguV3InstallOrChangeUseCase(t)
			if enabled {
				handler.EXPECT().HandleUntilApplied(testCtx, mock.MatchedBy(func(dogu *v3beta1.Dogu) bool {
					return dogu.Name == "modern" && dogu.Namespace == testNamespace &&
						dogu.Spec.Name == "modern" && dogu.Spec.DoguNamespace == "official" && dogu.Spec.DoguApiVersion == "v3"
				})).Return(0, nil).Once()
			}
			cl := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(&modern).Build()
			dispatcher := &DoguReconciler{client: cl, doguV3Enabled: enabled, doguV3ChangeHandler: handler}
			_, err = dispatcher.Reconcile(testCtx, controllerruntime.Request{
				NamespacedName: types.NamespacedName{Name: modernEvent.Object.Name, Namespace: modernEvent.Object.Namespace},
			})
			require.NoError(t, err)
		})
	}
}

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
