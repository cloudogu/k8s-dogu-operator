package manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeploymentManager(t *testing.T) {
	t.Run("Successfully created deployment manager", func(t *testing.T) {
		manager := NewDeploymentManager(
			newMockPodInterface(t),
			newMockDeploymentInterface(t),
		)

		assert.NotNil(t, manager)
	})
}

func Test_deploymentManager_GetLastStartingTime(t *testing.T) {
	type fields struct {
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
		podInterfaceFn        func(t *testing.T) podInterface
	}
	tests := []struct {
		name           string
		fields         fields
		deploymentName string
		want           *time.Time
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			deploymentName: "test",
			want:           nil,
			wantErr:        assert.Error,
		},
		{
			name: "should fail to get pods of deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Selector: &metav1.LabelSelector{},
						},
					}, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					mck := newMockPodInterface(t)
					labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{})
					mck.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: labelSelector}).Return(nil, assert.AnError)
					return mck
				},
			},
			deploymentName: "test",
			want:           nil,
			wantErr:        assert.Error,
		},
		{
			name: "should succeed get last starting time of deployment",
			fields: fields{
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{
						Spec: appsv1.DeploymentSpec{
							Selector: &metav1.LabelSelector{},
						},
					}, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					mck := newMockPodInterface(t)
					labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{})
					pods := &v1.PodList{Items: []v1.Pod{
						{
							Status: v1.PodStatus{
								StartTime: &metav1.Time{},
							},
						},
					}}
					mck.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: labelSelector}).Return(pods, nil)
					return mck
				},
			},
			deploymentName: "test",
			want:           &time.Time{},
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &deploymentManager{
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				podInterface:        tt.fields.podInterfaceFn(t),
			}
			got, err := dm.GetLastStartingTime(testCtx, tt.deploymentName)
			if !tt.wantErr(t, err, fmt.Sprintf("GetLastStartingTime(%v, %v)", testCtx, tt.deploymentName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLastStartingTime(%v, %v)", testCtx, tt.deploymentName)
		})
	}
}
