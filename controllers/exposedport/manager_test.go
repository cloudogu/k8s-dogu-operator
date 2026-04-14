package exposedport

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

//go:embed testdata/k8s-ces-gateway-config-expected-for-add.yaml
var expectedAddConfigMapBytes []byte

//go:embed testdata/k8s-ces-gateway-config-expected-for-delete.yaml
var expectedDeleteConfigMapBytes []byte

//go:embed testdata/k8s-ces-gateway-config-previous.yaml
var previousConfigMapBytes []byte

//go:embed testdata/initial-exposed-ports-configmap.yaml
var initialExposedPortsConfigMapBytes []byte

//go:embed testdata/created-initial-exposed-ports-configmap.yaml
var createdInitialExposedPortsConfigMapBytes []byte

var testCtx = context.Background()

func readExposedPortsExpectedAddConfigMap(t *testing.T) *v1.ConfigMap {
	t.Helper()

	data := &v1.ConfigMap{}
	err := yaml.Unmarshal(expectedAddConfigMapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readInitialExposedPortsAddConfigMap(t *testing.T) *v1.ConfigMap {
	t.Helper()

	data := &v1.ConfigMap{}
	err := yaml.Unmarshal(initialExposedPortsConfigMapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readCreatedInitialExposedPortsAddConfigMap(t *testing.T) *v1.ConfigMap {
	t.Helper()

	data := &v1.ConfigMap{}
	err := yaml.Unmarshal(createdInitialExposedPortsConfigMapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readExposedPortsExpectedDeleteConfigMap(t *testing.T) *v1.ConfigMap {
	t.Helper()

	data := &v1.ConfigMap{}
	err := yaml.Unmarshal(expectedDeleteConfigMapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func readExposedPortsPreviousConfigMap(t *testing.T) *v1.ConfigMap {
	t.Helper()

	data := &v1.ConfigMap{}
	err := yaml.Unmarshal(previousConfigMapBytes, data)
	if err != nil {
		t.Fatal(err.Error())
	}

	return data
}

func Test_exposedPortsManager_AddPorts(t *testing.T) {
	type fields struct {
		configMapInterfaceFn func(t *testing.T) configMapInterface
	}
	type args struct {
		ports []core.ExposedPort
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1.ConfigMap
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should do nothing when no exposed ports exist",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					return newMockConfigMapInterface(t)
				},
			},
			args: args{
				ports: make([]core.ExposedPort, 0),
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "should add port to config map and create config map if does not exist",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					err := errors.NewNotFound(schema.GroupResource{
						Group:    "",
						Resource: "configmaps",
					}, exposedPortsConfigMapName)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(nil, err)
					mck.EXPECT().Get(testCtx, initialExposedPortsConfigMapName, metav1.GetOptions{}).Return(readInitialExposedPortsAddConfigMap(t), nil)
					mck.EXPECT().Create(testCtx, readCreatedInitialExposedPortsAddConfigMap(t), metav1.CreateOptions{}).Return(readCreatedInitialExposedPortsAddConfigMap(t), nil)
					mck.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(readExposedPortsExpectedAddConfigMap(t), nil)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 10,
						Host:      10,
					},
				},
			},
			wantErr: assert.NoError,
			want:    readExposedPortsExpectedAddConfigMap(t),
		},
		{
			name: "should fail to add port to config map because configmap could not be updated",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(readExposedPortsPreviousConfigMap(t), nil)
					mck.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 10,
						Host:      10,
					},
				},
			},
			wantErr: assert.Error,
			want:    nil,
		},
		{
			name: "should succeed to add port to config map",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(readExposedPortsPreviousConfigMap(t), nil)
					mck.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(readExposedPortsExpectedAddConfigMap(t), nil)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 10,
						Host:      10,
					},
				},
			},
			wantErr: assert.NoError,
			want:    readExposedPortsExpectedAddConfigMap(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epm := &exposedPortsManager{
				configMapInterface: tt.fields.configMapInterfaceFn(t),
			}
			got, err := epm.AddPorts(testCtx, tt.args.ports)
			if !tt.wantErr(t, err, fmt.Sprintf("AddPorts(%v, %v)", testCtx, tt.args.ports)) {
				return
			}
			if got != nil {
				assert.Equalf(t, tt.want.Data, got.Data, "AddPorts(%v, %v)", testCtx, tt.args.ports)
			} else {
				assert.Equalf(t, tt.want, got, "AddPorts(%v, %v)", testCtx, tt.args.ports)
			}
		})
	}
}

func Test_exposedPortsManager_DeletePorts(t *testing.T) {
	type fields struct {
		configMapInterfaceFn func(t *testing.T) configMapInterface
	}
	type args struct {
		ports []core.ExposedPort
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1.ConfigMap
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should do nothing when no exposed ports exist",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					return newMockConfigMapInterface(t)
				},
			},
			args: args{
				ports: make([]core.ExposedPort, 0),
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "should fail to delete port out of config map because configmap does not exist",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 2222,
						Host:      2222,
					},
				},
			},
			wantErr: assert.Error,
			want:    nil,
		},
		{
			name: "should fail to delete port out of config map because configmap could not be updated",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(readExposedPortsPreviousConfigMap(t), nil)
					mck.EXPECT().Update(testCtx, readExposedPortsExpectedDeleteConfigMap(t), metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 2222,
						Host:      2222,
					},
				},
			},
			wantErr: assert.Error,
			want:    nil,
		},
		{
			name: "should succeed to delete port out of config map",
			fields: fields{
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, exposedPortsConfigMapName, metav1.GetOptions{}).Return(readExposedPortsPreviousConfigMap(t), nil)
					mck.EXPECT().Update(testCtx, mock.Anything, metav1.UpdateOptions{}).Return(readExposedPortsExpectedDeleteConfigMap(t), nil)
					return mck
				},
			},
			args: args{
				ports: []core.ExposedPort{
					{
						Type:      "tcp",
						Container: 2222,
						Host:      2222,
					},
				},
			},
			wantErr: assert.NoError,
			want:    readExposedPortsExpectedDeleteConfigMap(t),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epm := &exposedPortsManager{
				configMapInterface: tt.fields.configMapInterfaceFn(t),
			}
			got, err := epm.DeletePorts(testCtx, tt.args.ports)
			if !tt.wantErr(t, err, fmt.Sprintf("DeletePorts(%v, %v)", testCtx, tt.args.ports)) {
				return
			}
			if got != nil {
				assert.Equalf(t, tt.want.Data, got.Data, "DeletePorts(%v, %v)", testCtx, tt.args.ports)
			} else {
				assert.Equalf(t, tt.want, got, "DeletePorts(%v, %v)", testCtx, tt.args.ports)
			}
		})
	}
}
