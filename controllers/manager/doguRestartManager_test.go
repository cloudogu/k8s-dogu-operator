package manager

import (
	"fmt"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguRestartManager(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDoguRestartManager(
			newMockDoguInterface(t),
			newMockDeploymentInterface(t),
		)

		assert.NotNil(t, step)
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
					mck.EXPECT().Update(testCtx, &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v3.PodTemplateSpec{
								ObjectMeta: v1.ObjectMeta{
									Annotations: map[string]string{
										restartedAtAnnotationKey: time.Now().Format(time.RFC3339),
									},
								},
							},
						},
					}, v1.UpdateOptions{}).Return(nil, assert.AnError)
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
					mck.EXPECT().Update(testCtx, &appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Template: v3.PodTemplateSpec{
								ObjectMeta: v1.ObjectMeta{
									Annotations: map[string]string{
										restartedAtAnnotationKey: time.Now().Format(time.RFC3339),
									},
								},
							},
						},
					}, v1.UpdateOptions{}).Return(nil, nil)
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
