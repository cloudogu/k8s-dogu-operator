package manager

import (
	"context"
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguRestartManager(t *testing.T) {
	t.Run("Successfully created restart manager", func(t *testing.T) {
		doguMock := newMockDoguInterface(t)
		deploymentMock := newMockDeploymentInterface(t)
		step := NewDoguRestartManager(
			doguMock,
			deploymentMock,
		)

		assert.Same(t, doguMock, step.(*doguRestartManager).doguInterface)
		assert.Same(t, deploymentMock, step.(*doguRestartManager).deploymentInterface)
	})
}

func Test_doguRestartManager_RestartDogu(t *testing.T) {
	type fields struct {
		doguInterfaceFn       func(t *testing.T) doguInterface
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
	}
	tests := []struct {
		name    string
		fields  fields
		dogu    *v2.Dogu
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get deployment of dogu",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			wantErr: assert.Error,
		},
		{
			name: "should fail to update deployment of dogu",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					mck.EXPECT().Update(testCtx, mock.Anything, v1.UpdateOptions{}).
						Run(func(ctx context.Context, deployment *appsv1.Deployment, opts v1.UpdateOptions) {
							restartedAt, exists := deployment.Spec.Template.Annotations[restartedAtAnnotationKey]
							assert.True(t, exists, "restartedAt annotation should exist")
							_, err := time.Parse(time.RFC3339, restartedAt)
							assert.NoError(t, err, "restartedAt should be formatted correctly")
						}).
						Return(nil, assert.AnError)
					return mck
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			wantErr: assert.Error,
		},
		{
			name: "should succeed to update deployment of dogu",
			fields: fields{
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					mck.EXPECT().Update(testCtx, mock.Anything, v1.UpdateOptions{}).
						Run(func(ctx context.Context, deployment *appsv1.Deployment, opts v1.UpdateOptions) {
							restartedAt, exists := deployment.Spec.Template.Annotations[restartedAtAnnotationKey]
							assert.True(t, exists, "restartedAt annotation should exist")
							_, err := time.Parse(time.RFC3339, restartedAt)
							assert.NoError(t, err, "restartedAt should be formatted correctly")
						}).
						Return(nil, nil)
					return mck
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drm := &doguRestartManager{
				doguInterface:       tt.fields.doguInterfaceFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
			}
			tt.wantErr(t, drm.RestartDogu(testCtx, tt.dogu), fmt.Sprintf("RestartDogu(%v, %v)", testCtx, tt.dogu))
		})
	}
}
