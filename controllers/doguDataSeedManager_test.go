package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func Test_doguDataSeedManager_DataMountsChanged(t *testing.T) {
	nginxDeploymentWithOutdatedSeederMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dogu-data-seeder-init",
							Image: "",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "customhtml",
									MountPath: "/dogumount/customhtml",
									SubPath:   "customhtml",
								},
								{
									Name:      "oldconfigmap",
									MountPath: "/datamount/oldconfigmap",
									SubPath:   "oldconfigmap",
								},
							},
						},
					},
				},
			},
		},
	}

	nginxDeploymentWithoutDataSeederMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{},
				},
			},
		},
	}

	nginxDeploymentWithSameSeederMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dogu-data-seeder-init",
							Image: "",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "customhtml",
									MountPath: "/dogumount/customhtml",
									SubPath:   "customhtml",
								},
								{
									Name:      "configmap",
									MountPath: "/datamount/configmap",
									SubPath:   "configmap",
								},
							},
						},
					},
				},
			},
		},
	}

	nginxDoguResourceWithSeederMounts := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Name:      "nginx",
			Namespace: testNamespace,
		},
		Spec: v2.DoguSpec{
			Data: []v2.DataMount{
				{
					SourceType: "ConfigMap",
					Name:       "configmap",
					Volume:     "customhtml",
				},
			},
		},
	}

	expectedInitContainerWithSeederMounts := &corev1.Container{
		Name:  "dogu-data-seeder-init",
		Image: "",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "customhtml",
				MountPath: "/dogumount/customhtml",
				SubPath:   "customhtml",
			},
			{
				Name:      "configmap",
				MountPath: "/datamount/configmap",
				SubPath:   "configmap",
			},
		},
	}

	nginxDogu := &core.Dogu{}

	type fields struct {
		deploymentInterface func() deploymentInterface
		resourceGenerator   func() dataSeederInitContainerGenerator
		resourceDoguFetcher func() resourceDoguFetcher
	}
	type args struct {
		ctx          context.Context
		doguResource *v2.Dogu
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return true if data mounts differ from deployment entries",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedSeederMounts}}, nil)
					return mock
				},
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResourceWithSeederMounts).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResourceWithSeederMounts, "").Return(expectedInitContainerWithSeederMounts, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should return true if deployment has not init container yet",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithoutDataSeederMounts}}, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should return false if init container mounts matches mounts in dogu cr",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithSameSeederMounts}}, nil)
					return mock
				},
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResourceWithSeederMounts).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResourceWithSeederMounts, "").Return(expectedInitContainerWithSeederMounts, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			want:    false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguDataSeedManager{}
			if tt.fields.deploymentInterface != nil {
				m.deploymentInterface = tt.fields.deploymentInterface()
			}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.resourceDoguFetcher != nil {
				m.resourceDoguFetcher = tt.fields.resourceDoguFetcher()
			}

			got, err := m.DataMountsChanged(tt.args.ctx, tt.args.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("DataMountsChanged(%v, %v)", tt.args.ctx, tt.args.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "DataMountsChanged(%v, %v)", tt.args.ctx, tt.args.doguResource)
		})
	}
}

func Test_doguDataSeedManager_getDoguDeployment(t *testing.T) {
	nginxDoguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "nginx"}}

	type fields struct {
		deploymentInterface func() deploymentInterface
	}
	type args struct {
		ctx          context.Context
		doguResource *v2.Dogu
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *appsv1.Deployment
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return error on get deployment error",
			fields: fields{deploymentInterface: func() deploymentInterface {
				mock := newMockDeploymentInterface(t)
				mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(nil, assert.AnError)
				return mock
			}},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResource,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to get deployment for dogu nginx")
				return true
			},
		},
		{
			name: "should return error on invalid amount of dogu deployments",
			fields: fields{deploymentInterface: func() deploymentInterface {
				mock := newMockDeploymentInterface(t)
				mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{}}, nil)
				return mock
			}},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResource,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorContains(t, err, "dogu nginx has more than one or zero deployments")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguDataSeedManager{}
			if tt.fields.deploymentInterface != nil {
				m.deploymentInterface = tt.fields.deploymentInterface()
			}
			got, err := m.getDoguDeployment(tt.args.ctx, tt.args.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("getDoguDeployment(%v, %v)", tt.args.ctx, tt.args.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getDoguDeployment(%v, %v)", tt.args.ctx, tt.args.doguResource)
		})
	}
}

func Test_doguDataSeedManager_createDataMountInitContainer(t *testing.T) {
	nginxDoguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "nginx"}}
	nginxDogu := &core.Dogu{Name: "nginx"}

	type fields struct {
		resourceGenerator   func() dataSeederInitContainerGenerator
		resourceDoguFetcher func() resourceDoguFetcher
	}
	type args struct {
		ctx          context.Context
		doguResource *v2.Dogu
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.Container
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return error on error fetching dogu resource",
			fields: fields{
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResource).Return(nil, nil, assert.AnError)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResource,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorContains(t, err, "failed to get dogu descriptor for dogu nginx")
				return true
			},
		},
		{
			name: "should return error on error getting data seed container",
			fields: fields{
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResource).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResource, "").Return(nil, assert.AnError)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResource,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorContains(t, err, "failed to generate data seeder init container while diff calculation")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguDataSeedManager{}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.resourceDoguFetcher != nil {
				m.resourceDoguFetcher = tt.fields.resourceDoguFetcher()
			}
			got, err := m.createDataMountInitContainer(tt.args.ctx, tt.args.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("createDataMountInitContainer(%v, %v)", tt.args.ctx, tt.args.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "createDataMountInitContainer(%v, %v)", tt.args.ctx, tt.args.doguResource)
		})
	}
}

func TestNewDoguDataSeedManager(t *testing.T) {
	t.Run("should set attributes", func(t *testing.T) {
		// given
		deploymentMock := newMockDeploymentInterface(t)
		resourceGeneratorMock := newMockDataSeederInitContainerGenerator(t)
		resourceDoguFetcherMock := newMockResourceDoguFetcher(t)

		// when
		sut := NewDoguDataSeedManager(deploymentMock, resourceGeneratorMock, resourceDoguFetcherMock, map[string]string{config.DataSeederImageConfigmapNameKey: "image"})

		// then
		require.NotNil(t, sut)
		assert.Equal(t, deploymentMock, sut.deploymentInterface)
		assert.Equal(t, resourceGeneratorMock, sut.resourceGenerator)
		assert.Equal(t, resourceDoguFetcherMock, sut.resourceDoguFetcher)
		assert.Equal(t, "image", sut.image)
	})
}

func Test_doguDataSeedManager_UpdateDataMounts(t *testing.T) {
	nginxDeploymentWithOutdatedSeederMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "others should be kept",
						},
						{
							Name:  "dogu-data-seeder-init",
							Image: "",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "customhtml",
									MountPath: "/dogumount/customhtml",
									SubPath:   "customhtml",
								},
								{
									Name:      "oldconfigmap",
									MountPath: "/datamount/oldconfigmap",
									SubPath:   "oldconfigmap",
								},
							},
						},
					},
				},
			},
		},
	}

	updatedInitContainer := &corev1.Container{
		Name:  "dogu-data-seeder-init",
		Image: "",
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "customhtml",
				MountPath: "/dogumount/customhtml",
				SubPath:   "customhtml",
			},
			{
				Name:      "configmap",
				MountPath: "/datamount/configmap",
				SubPath:   "configmap",
			},
		},
	}

	nginxDeploymentWithNewSeederMounts := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "others should be kept",
						},
						*updatedInitContainer,
					},
				},
			},
		},
	}

	nginxDoguResourceWithSeederMounts := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Name:      "nginx",
			Namespace: testNamespace,
		},
		Spec: v2.DoguSpec{
			Data: []v2.DataMount{
				{
					SourceType: "ConfigMap",
					Name:       "configmap",
					Volume:     "customhtml",
				},
			},
		},
	}

	nginxDogu := &core.Dogu{}

	type fields struct {
		deploymentInterface func() deploymentInterface
		resourceGenerator   func() dataSeederInitContainerGenerator
		resourceDoguFetcher func() resourceDoguFetcher
	}
	type args struct {
		ctx          context.Context
		doguResource *v2.Dogu
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should update existing init container",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedSeederMounts}}, nil)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewSeederMounts, v1.UpdateOptions{}).Return(nil, nil)
					return mock
				},
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResourceWithSeederMounts).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResourceWithSeederMounts, "").Return(updatedInitContainer, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should should retry on conflict error",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedSeederMounts}}, nil).Times(2)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewSeederMounts, v1.UpdateOptions{}).Return(nil, errors.NewConflict(schema.GroupResource{}, "name", assert.AnError)).Times(1)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewSeederMounts, v1.UpdateOptions{}).Return(nil, nil).Times(1)
					return mock
				},
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResourceWithSeederMounts).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResourceWithSeederMounts, "").Return(updatedInitContainer, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return retry error on general error",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(nil, assert.AnError)
					return mock
				},
				resourceDoguFetcher: func() resourceDoguFetcher {
					mock := newMockResourceDoguFetcher(t)
					mock.EXPECT().FetchWithResource(testCtx, nginxDoguResourceWithSeederMounts).Return(nginxDogu, nil, nil)
					return mock
				},
				resourceGenerator: func() dataSeederInitContainerGenerator {
					mock := newMockDataSeederInitContainerGenerator(t)
					mock.EXPECT().GetDataSeederContainer(nginxDogu, nginxDoguResourceWithSeederMounts, "").Return(updatedInitContainer, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithSeederMounts,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to update deployment dogu data mount for dogu nginx")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguDataSeedManager{}
			if tt.fields.deploymentInterface != nil {
				m.deploymentInterface = tt.fields.deploymentInterface()
			}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.resourceDoguFetcher != nil {
				m.resourceDoguFetcher = tt.fields.resourceDoguFetcher()
			}
			tt.wantErr(t, m.UpdateDataMounts(tt.args.ctx, tt.args.doguResource), fmt.Sprintf("UpdateDataMounts(%v, %v)", tt.args.ctx, tt.args.doguResource))
		})
	}
}
