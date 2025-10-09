package manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const testNamespace = "ecosystem"

var testCtx = context.Background()

func Test_doguAdditionalMountsManager_AdditionalMountsChanged(t *testing.T) {
	nginxDeploymentWithOutdatedAdditionalMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dogu-additional-mounts-init",
							Image: "",
							Args: []string{
								"copy",
								"-source=/datamount/oldconfigmap",
								"-target=/dogumount/customhtml",
							},
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

	nginxDeploymentWithoutAdditionalMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{},
				},
			},
		},
	}

	nginxDeploymentWithSameAdditionalMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "dogu-additional-mounts-init",
							Image: "",
							Args: []string{
								"copy",
								"-source=/datamount/configmap",
								"-target=/dogumount/customhtml",
							},
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

	nginxDoguResourceWithAdditionalMounts := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Name:      "nginx",
			Namespace: testNamespace,
		},
		Spec: v2.DoguSpec{
			AdditionalMounts: []v2.DataMount{
				{
					SourceType: "ConfigMap",
					Name:       "configmap",
					Volume:     "customhtml",
				},
			},
		},
	}

	expectedInitContainerWithAdditionalMounts := &corev1.Container{
		Name:  "dogu-additional-mounts-init",
		Image: "",
		Args: []string{
			"copy",
			"-source=/datamount/configmap",
			"-target=/dogumount/customhtml",
		},
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
		deploymentInterface   func() deploymentInterface
		resourceGenerator     func() resourceGenerator
		localDoguFetcher      func() localDoguFetcher
		requirementsGenerator func() requirementsGenerator
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
			name: "should return true if additional mounts differ from deployment entries",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedAdditionalMounts}}, nil)
					return mock
				},
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts, "", corev1.ResourceRequirements{}).Return(expectedInitContainerWithAdditionalMounts, nil)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should return true if deployment has not init container yet",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithoutAdditionalMounts}}, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should return false if init container mounts matches mounts in dogu cr",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithSameAdditionalMounts}}, nil)
					return mock
				},
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts, "", corev1.ResourceRequirements{}).Return(expectedInitContainerWithAdditionalMounts, nil)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			want:    false,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguAdditionalMountManager{}
			if tt.fields.deploymentInterface != nil {
				m.deploymentInterface = tt.fields.deploymentInterface()
			}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.localDoguFetcher != nil {
				m.localDoguFetcher = tt.fields.localDoguFetcher()
			}
			if tt.fields.requirementsGenerator != nil {
				m.requirementsGenerator = tt.fields.requirementsGenerator()
			}

			got, err := m.AdditionalMountsChanged(tt.args.ctx, tt.args.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("AdditionalMountsChanged(%v, %v)", tt.args.ctx, tt.args.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "AdditionalMountsChanged(%v, %v)", tt.args.ctx, tt.args.doguResource)
		})
	}
}

func Test_doguAdditionalMountsManager_getDoguDeployment(t *testing.T) {
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
			m := &doguAdditionalMountManager{}
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

func Test_doguAdditionalMountsManager_createDataMountInitContainer(t *testing.T) {
	nginxDoguResource := &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "nginx"}}
	nginxDogu := &core.Dogu{Name: "nginx"}

	type fields struct {
		resourceGenerator     func() resourceGenerator
		localDoguFetcher      func() localDoguFetcher
		requirementsGenerator func() requirementsGenerator
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
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResource.Name)).Return(nil, assert.AnError)
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
			name: "should return error on error getting additional mounts container",
			fields: fields{
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResource.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResource, "", corev1.ResourceRequirements{}).Return(nil, assert.AnError)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResource,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorContains(t, err, "failed to generate dogu additional mounts init container while diff calculation")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguAdditionalMountManager{}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.localDoguFetcher != nil {
				m.localDoguFetcher = tt.fields.localDoguFetcher()
			}
			if tt.fields.requirementsGenerator != nil {
				m.requirementsGenerator = tt.fields.requirementsGenerator()
			}
			got, err := m.createAdditionalMountInitContainer(tt.args.ctx, tt.args.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("createAdditionalMountInitContainer(%v, %v)", tt.args.ctx, tt.args.doguResource)) {
				return
			}
			assert.Equalf(t, tt.want, got, "createAdditionalMountInitContainer(%v, %v)", tt.args.ctx, tt.args.doguResource)
		})
	}
}

func TestNewdoguAdditionalMountsManager(t *testing.T) {
	t.Run("should set attributes", func(t *testing.T) {
		// given
		deploymentMock := newMockDeploymentInterface(t)
		resourceGeneratorMock := newMockResourceGenerator(t)
		localDoguFetcherMock := newMockLocalDoguFetcher(t)
		requirementsGeneratorMock := newMockRequirementsGenerator(t)
		mountsValidatorMock := newMockDoguAdditionalMountsValidator(t)
		images := resource.AdditionalImages{config.AdditionalMountsInitContainerImageConfigmapNameKey: "image"}

		// when
		sut := NewDoguAdditionalMountManager(deploymentMock, resourceGeneratorMock, localDoguFetcherMock, requirementsGeneratorMock, mountsValidatorMock, images)

		// then
		require.NotNil(t, sut)
		assert.Equal(t, deploymentMock, sut.(*doguAdditionalMountManager).deploymentInterface)
		assert.Equal(t, resourceGeneratorMock, sut.(*doguAdditionalMountManager).resourceGenerator)
		assert.Equal(t, localDoguFetcherMock, sut.(*doguAdditionalMountManager).localDoguFetcher)
		assert.Equal(t, "image", sut.(*doguAdditionalMountManager).image)
	})
}

func Test_doguAdditionalMountsManager_UpdateAdditionalMounts(t *testing.T) {
	nginxDeploymentWithOutdatedAdditionalMounts := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "others should be kept",
						},
						{
							Name:  "dogu-additional-mounts-init",
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
		Name:  "dogu-additional-mounts-init",
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

	nginxDeploymentWithNewAdditionalMounts := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name: "others should be kept",
						},
						*updatedInitContainer,
					},
					Volumes: CreateExpectedVolumes(),
				},
			},
		},
	}

	nginxDoguResourceWithAdditionalMounts := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Name:      "nginx",
			Namespace: testNamespace,
		},
		Spec: v2.DoguSpec{
			AdditionalMounts: []v2.DataMount{
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
		deploymentInterface       func() deploymentInterface
		resourceGenerator         func() resourceGenerator
		localDoguFetcher          func() localDoguFetcher
		requirementsGenerator     func() requirementsGenerator
		additionalMountsValidator func() doguAdditionalMountsValidator
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
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedAdditionalMounts}}, nil)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewAdditionalMounts, v1.UpdateOptions{}).Return(nil, nil)
					return mock
				},
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts, "", corev1.ResourceRequirements{}).Return(updatedInitContainer, nil)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
				additionalMountsValidator: func() doguAdditionalMountsValidator {
					mock := newMockDoguAdditionalMountsValidator(t)
					mock.EXPECT().ValidateAdditionalMounts(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should should retry on conflict error",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					mock.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: "dogu.name=nginx"}).Return(&appsv1.DeploymentList{Items: []appsv1.Deployment{nginxDeploymentWithOutdatedAdditionalMounts}}, nil).Times(2)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewAdditionalMounts, v1.UpdateOptions{}).Return(nil, errors.NewConflict(schema.GroupResource{}, "name", assert.AnError)).Times(1)
					mock.EXPECT().Update(testCtx, nginxDeploymentWithNewAdditionalMounts, v1.UpdateOptions{}).Return(nil, nil).Times(1)
					return mock
				},
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts, "", corev1.ResourceRequirements{}).Return(updatedInitContainer, nil)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
				additionalMountsValidator: func() doguAdditionalMountsValidator {
					mock := newMockDoguAdditionalMountsValidator(t)
					mock.EXPECT().ValidateAdditionalMounts(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
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
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					mock.EXPECT().BuildAdditionalMountInitContainer(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts, "", corev1.ResourceRequirements{}).Return(updatedInitContainer, nil)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					mock.EXPECT().Generate(testCtx, nginxDogu).Return(corev1.ResourceRequirements{}, nil)
					return mock
				},
				additionalMountsValidator: func() doguAdditionalMountsValidator {
					mock := newMockDoguAdditionalMountsValidator(t)
					mock.EXPECT().ValidateAdditionalMounts(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to update deployment additional mounts for dogu nginx")
				return true
			},
		},
		{
			name: "should return validation error on failing validation",
			fields: fields{
				deploymentInterface: func() deploymentInterface {
					mock := newMockDeploymentInterface(t)
					return mock
				},
				localDoguFetcher: func() localDoguFetcher {
					mock := newMockLocalDoguFetcher(t)
					mock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(nginxDoguResourceWithAdditionalMounts.Name)).Return(nginxDogu, nil)
					return mock
				},
				resourceGenerator: func() resourceGenerator {
					mock := newMockResourceGenerator(t)
					return mock
				},
				requirementsGenerator: func() requirementsGenerator {
					mock := newMockRequirementsGenerator(t)
					return mock
				},
				additionalMountsValidator: func() doguAdditionalMountsValidator {
					mock := newMockDoguAdditionalMountsValidator(t)
					mock.EXPECT().ValidateAdditionalMounts(testCtx, nginxDogu, nginxDoguResourceWithAdditionalMounts).Return(assert.AnError)
					return mock
				},
			},
			args: args{
				ctx:          testCtx,
				doguResource: nginxDoguResourceWithAdditionalMounts,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "additional mounts are not valid for dogu nginx")
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &doguAdditionalMountManager{}
			if tt.fields.deploymentInterface != nil {
				m.deploymentInterface = tt.fields.deploymentInterface()
			}
			if tt.fields.resourceGenerator != nil {
				m.resourceGenerator = tt.fields.resourceGenerator()
			}
			if tt.fields.localDoguFetcher != nil {
				m.localDoguFetcher = tt.fields.localDoguFetcher()
			}
			if tt.fields.requirementsGenerator != nil {
				m.requirementsGenerator = tt.fields.requirementsGenerator()
			}
			if tt.fields.additionalMountsValidator != nil {
				m.doguAdditionalMountValidator = tt.fields.additionalMountsValidator()
			}
			tt.wantErr(t, m.UpdateAdditionalMounts(tt.args.ctx, tt.args.doguResource), fmt.Sprintf("UpdateAdditionalMounts(%v, %v)", tt.args.ctx, tt.args.doguResource))
		})
	}
}

// CreateExpectedVolumes creates a set of volumes that match the expected volumes in the deployment
func CreateExpectedVolumes() []corev1.Volume {
	optional := true
	return []corev1.Volume{
		{
			Name: "dogu-health",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "k8s-dogu-operator-dogu-health",
					},
				},
			},
		},
		{
			Name: "nginx-ephemeral",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "global-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "global-config",
					},
				},
			},
		},
		{
			Name: "normal-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "nginx-config",
					},
				},
			},
		},
		{
			Name: "sensitive-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "nginx-config",
				},
			},
		},
		{
			Name: "-dogu-json",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "dogu-spec-",
					},
					Optional: &optional,
				},
			},
		},
		{
			Name: "configmap",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "configmap",
					},
				},
			},
		},
	}
}
