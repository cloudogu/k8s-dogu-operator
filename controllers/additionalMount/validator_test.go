package additionalMount

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func TestValidator_ValidateAdditionalMounts(t *testing.T) {
	testCtx := context.Background()

	nginxDogu := &core.Dogu{
		Volumes: []core.Volume{
			{
				Name: "customhtml",
				Path: "/var/www/customhtml",
			},
			{
				Name: "app.conf.d",
				Path: "/etc/nginx/app.conf.d",
			},
			{
				Name: "localConfig",
				Path: "/var/ces/config",
			},
			{
				Name: "menu-json",
				Path: "/var/www/html/warp/menu",
				Clients: []core.VolumeClient{
					{
						Name: "k8s-dogu-operator",
					},
				},
			},
		},
	}

	validDoguAdditionalMounts := &k8sv2.Dogu{
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "customhtml",
				},
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "customhtml",
					Subfolder:  "bc",
				},
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap2",
					Volume:     "customhtml",
				},
				{
					SourceType: k8sv2.DataSourceSecret,
					Name:       "secret",
					Volume:     "app.conf.d",
				},
			},
		},
	}

	duplicatedDoguAdditionalMounts := &k8sv2.Dogu{
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "customhtml",
				},
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "customhtml",
				},
			},
		},
	}

	notFoundDoguAdditionalMount := &k8sv2.Dogu{
		ObjectMeta: v1.ObjectMeta{Name: "nginx"},
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "not found",
				},
			},
		},
	}

	volumeClientAdditionalMount := &k8sv2.Dogu{
		ObjectMeta: v1.ObjectMeta{Name: "nginx"},
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "menu-json",
				},
			},
		},
	}

	simpleDoguAdditionalMounts := &k8sv2.Dogu{
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: k8sv2.DataSourceConfigMap,
					Name:       "configmap1",
					Volume:     "customhtml",
				},
			},
		},
	}

	invalidSourceTypeDoguAdditionalMounts := &k8sv2.Dogu{
		Spec: k8sv2.DoguSpec{
			Name: "nginx",
			AdditionalMounts: []k8sv2.DataMount{
				{
					SourceType: "invalid",
					Name:       "configmap1",
					Volume:     "customhtml",
				},
			},
		},
	}

	type fields struct {
		configMapInterface func(t *testing.T) configMapGetter
		secretInterface    func(t *testing.T) secretGetter
	}
	type args struct {
		ctx            context.Context
		doguDescriptor *core.Dogu
		doguResource   *k8sv2.Dogu
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		assertError func(t assert.TestingT, err error)
	}{
		{
			name: "should succeed with multiple mounts in one volume and the same mount in different volumes",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(2).Return(nil, nil)
					mock.EXPECT().Get(testCtx, "configmap2", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
				secretInterface: func(t *testing.T) secretGetter {
					mock := newMockSecretGetter(t)
					mock.EXPECT().Get(testCtx, "secret", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguDescriptor: nginxDogu,
				doguResource:   validDoguAdditionalMounts,
			},
			wantErr: false,
		},
		{
			name: "should return an error on duplicated additional mounts entries",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguResource:   duplicatedDoguAdditionalMounts,
				doguDescriptor: nginxDogu,
			},
			wantErr: true,
			assertError: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "duplicate entry")
			},
		},
		{
			name: "should return an error on non existent dogu volume",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguResource:   notFoundDoguAdditionalMount,
				doguDescriptor: nginxDogu,
			},
			wantErr: true,
			assertError: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "volume not found does not exists in dogu descriptor for dogu nginx")
			},
		},
		{
			name: "should return an error on dogu volume with volume clients",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguResource:   volumeClientAdditionalMount,
				doguDescriptor: nginxDogu,
			},
			wantErr: true,
			assertError: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "volume menu-json with volumeclients is currently not supported for addtitionalMounts on dogu nginx")
			},
		},
		{
			name: "should not retry on not found error",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, errors.NewNotFound(schema.GroupResource{}, "configmap1"))

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguResource:   simpleDoguAdditionalMounts,
				doguDescriptor: nginxDogu,
			},
			wantErr: true,
			assertError: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "\"configmap1\" not found")
			},
		},
		{
			name: "should retry on other errors",
			fields: fields{
				configMapInterface: func(t *testing.T) configMapGetter {
					mock := newMockConfigMapGetter(t)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, assert.AnError)
					mock.EXPECT().Get(testCtx, "configmap1", v1.GetOptions{}).Times(1).Return(nil, nil)

					return mock
				},
			},
			args: args{
				ctx:            testCtx,
				doguResource:   simpleDoguAdditionalMounts,
				doguDescriptor: nginxDogu,
			},
			wantErr: false,
		},
		{
			name: "should return error on invalid source type",
			args: args{
				ctx:            testCtx,
				doguResource:   invalidSourceTypeDoguAdditionalMounts,
				doguDescriptor: nginxDogu,
			},
			wantErr: true,
			assertError: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "unknown additional mount type invalid for dogu")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var configMapMock configMapGetter
			if tt.fields.configMapInterface != nil {
				configMapMock = tt.fields.configMapInterface(t)
			}

			var secretMapMock secretGetter
			if tt.fields.secretInterface != nil {
				secretMapMock = tt.fields.secretInterface(t)
			}

			v := &Validator{
				configMapInterface: configMapMock,
				secretInterface:    secretMapMock,
			}
			err := v.ValidateAdditionalMounts(tt.args.ctx, tt.args.doguDescriptor, tt.args.doguResource)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAdditionalMounts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.assertError != nil {
				tt.assertError(t, err)
			}
		})
	}
}

func TestNewValidator(t *testing.T) {
	t.Run("should set parameter", func(t *testing.T) {
		// given
		configMapMock := newMockConfigMapGetter(t)
		secretMapMock := newMockSecretGetter(t)

		// when
		sut := NewValidator(configMapMock, secretMapMock)

		// then
		assert.Equal(t, configMapMock, sut.configMapInterface)
		assert.Equal(t, secretMapMock, sut.secretInterface)
	})
}
