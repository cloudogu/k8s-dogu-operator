package controllers

import (
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const testDoguName = "test"
const testNamespace = "ecosystem"

func TestNewGlobalConfigReconciler(t *testing.T) {
	// given
	restartManagerMock := newMockDoguRestartManager(t)
	configMapMock := newMockConfigMapInterface(t)
	doguInterfaceMock := newMockDoguInterface(t)
	managerMock := newMockCtrlManager(t)
	managerMock.EXPECT().GetControllerOptions().Return(config.Controller{})
	managerMock.EXPECT().GetScheme().Return(getTestScheme())
	managerMock.EXPECT().GetLogger().Return(logr.Logger{})
	managerMock.EXPECT().Add(mock.Anything).Return(nil)
	managerMock.EXPECT().GetCache().Return(nil)
	deploymentManagerMock := newMockDeploymentManager(t)

	// when
	reconciler, err := NewGlobalConfigReconciler(restartManagerMock, configMapMock, doguInterfaceMock, nil, managerMock, deploymentManagerMock)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, reconciler)
}

func TestGlobalConfigReconciler_Reconcile(t *testing.T) {
	type fields struct {
		doguRestartManagerFn func(t *testing.T) doguRestartManager
		configMapInterfaceFn func(t *testing.T) configMapInterface
		doguInterfaceFn      func(t *testing.T) doguInterface
		deploymentManagerFn  func(t *testing.T) deploymentManager
		doguEvents           chan<- event.TypedGenericEvent[*v2.Dogu]
	}
	tests := []struct {
		name    string
		fields  fields
		req     controllerruntime.Request
		want    controllerruntime.Result
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get config map",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					return newMockDeploymentManager(t)
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
		{
			name: "should fail to get config map",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					return newMockDeploymentManager(t)
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
		{
			name: "should fail to list dogus",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					cm := &coreV1.ConfigMap{
						ObjectMeta: v1.ObjectMeta{
							ManagedFields: []v1.ManagedFieldsEntry{
								{
									Time: &v1.Time{Time: time.Date(2025, 9, 24, 11, 1, 0, 0, &time.Location{})},
								},
							},
						},
					}
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(cm, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().List(testCtx, v1.ListOptions{}).Return(nil, assert.AnError)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					return newMockDeploymentManager(t)
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
		{
			name: "should fail to get last started time of deployment",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(&coreV1.ConfigMap{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogus := &v2.DoguList{
						Items: []v2.Dogu{
							{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{}).Return(dogus, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, testDoguName).Return(nil, assert.AnError)
					return mck
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.Error,
		},
		{
			name: "should continue if deployment is not yet found",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(&coreV1.ConfigMap{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogus := &v2.DoguList{
						Items: []v2.Dogu{
							{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{}).Return(dogus, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, testDoguName).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
				doguEvents: NewDoguEvents(),
			},
			want:    controllerruntime.Result{},
			wantErr: assert.NoError,
		},
		{
			name: "should fail to restart dogu",
			req:  controllerruntime.Request{NamespacedName: types.NamespacedName{Name: globalConfigMapName}},
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					t1 := time.Date(2025, 9, 24, 11, 1, 0, 0, &time.Location{})
					mck.EXPECT().Get(testCtx, globalConfigMapName, v1.GetOptions{}).Return(&coreV1.ConfigMap{
						ObjectMeta: v1.ObjectMeta{
							CreationTimestamp: v1.Time{Time: t1},
						},
					}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					dogus := &v2.DoguList{
						Items: []v2.Dogu{
							{ObjectMeta: v1.ObjectMeta{Name: testDoguName}},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{}).Return(dogus, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					mck := newMockDoguRestartManager(t)
					mck.EXPECT().RestartDogu(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: testDoguName},
					}).Return(assert.AnError)
					return mck
				},
				deploymentManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					t1 := time.Date(2025, 9, 24, 11, 0, 0, 0, &time.Location{})
					mck.EXPECT().GetLastStartingTime(testCtx, testDoguName).Return(&t1, nil)
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
				doguRestartManager: tt.fields.doguRestartManagerFn(t),
				configMapInterface: tt.fields.configMapInterfaceFn(t),
				doguInterface:      tt.fields.doguInterfaceFn(t),
				doguEvents:         tt.fields.doguEvents,
				deploymentManager:  tt.fields.deploymentManagerFn(t),
			}
			got, err := r.Reconcile(testCtx, tt.req)
			if !tt.wantErr(t, err, fmt.Sprintf("Reconcile(%v, %v)", testCtx, tt.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Reconcile(%v, %v)", testCtx, tt.req)
		})
	}
}
