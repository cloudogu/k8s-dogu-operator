package localregistry

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/client/v2"
	"testing"
)

func Test_etcdLocalDoguRegistry_Enable(t *testing.T) {
	tests := []struct {
		name           string
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
		dogu           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to enable",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Enable(testDoguLdap).Return(assert.AnError)
				return doguReg
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "should succeed to enable",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Enable(testDoguLdap).Return(nil)
				return doguReg
			},
			dogu:    testDoguLdap,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				doguRegistry: tt.doguRegistryFn(t),
			}
			tt.wantErr(t, er.Enable(testCtx, tt.dogu), fmt.Sprintf("Enable(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func Test_etcdLocalDoguRegistry_Register(t *testing.T) {
	tests := []struct {
		name           string
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
		dogu           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to register",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Register(testDoguLdap).Return(assert.AnError)
				return doguReg
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "should succeed to register",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Register(testDoguLdap).Return(nil)
				return doguReg
			},
			dogu:    testDoguLdap,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				doguRegistry: tt.doguRegistryFn(t),
			}
			tt.wantErr(t, er.Register(testCtx, tt.dogu), fmt.Sprintf("Register(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func Test_etcdLocalDoguRegistry_UnregisterAllVersions(t *testing.T) {
	type fields struct {
		registryFn     func(t *testing.T) registry.Registry
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
	}
	tests := []struct {
		name           string
		fields         fields
		simpleDoguName string
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to remove dogu config",
			fields: fields{
				registryFn: func(t *testing.T) registry.Registry {
					doguCfg := extMocks.NewConfigurationContext(t)
					doguCfg.EXPECT().RemoveAll().Return(assert.AnError)
					reg := extMocks.NewConfigurationRegistry(t)
					reg.EXPECT().DoguConfig("ldap").Return(doguCfg)
					return reg
				},
				doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
					doguReg := extMocks.NewDoguRegistry(t)
					return doguReg
				},
			},
			simpleDoguName: "ldap",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to remove dogu config for \"ldap\"", i)
			},
		},
		{
			name: "should fail to unregister dogu",
			fields: fields{
				registryFn: func(t *testing.T) registry.Registry {
					doguCfg := extMocks.NewConfigurationContext(t)
					doguCfg.EXPECT().RemoveAll().Return(nil)
					reg := extMocks.NewConfigurationRegistry(t)
					reg.EXPECT().DoguConfig("ldap").Return(doguCfg)
					return reg
				},
				doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
					doguReg := extMocks.NewDoguRegistry(t)
					doguReg.EXPECT().Unregister("ldap").Return(assert.AnError)
					return doguReg
				},
			},
			simpleDoguName: "ldap",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to unregister dogu \"ldap\"", i)
			},
		},
		{
			name: "should succeed if not found",
			fields: fields{
				registryFn: func(t *testing.T) registry.Registry {
					doguCfg := extMocks.NewConfigurationContext(t)
					doguCfg.EXPECT().RemoveAll().Return(client.Error{Code: client.ErrorCodeKeyNotFound})
					reg := extMocks.NewConfigurationRegistry(t)
					reg.EXPECT().DoguConfig("ldap").Return(doguCfg)
					return reg
				},
				doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
					doguReg := extMocks.NewDoguRegistry(t)
					doguReg.EXPECT().Unregister("ldap").Return(client.Error{Code: client.ErrorCodeKeyNotFound})
					return doguReg
				},
			},
			simpleDoguName: "ldap",
			wantErr:        assert.NoError,
		},
		{
			name: "should succeed",
			fields: fields{
				registryFn: func(t *testing.T) registry.Registry {
					doguCfg := extMocks.NewConfigurationContext(t)
					doguCfg.EXPECT().RemoveAll().Return(nil)
					reg := extMocks.NewConfigurationRegistry(t)
					reg.EXPECT().DoguConfig("ldap").Return(doguCfg)
					return reg
				},
				doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
					doguReg := extMocks.NewDoguRegistry(t)
					doguReg.EXPECT().Unregister("ldap").Return(nil)
					return doguReg
				},
			},
			simpleDoguName: "ldap",
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				registry:     tt.fields.registryFn(t),
				doguRegistry: tt.fields.doguRegistryFn(t),
			}
			tt.wantErr(t, er.UnregisterAllVersions(testCtx, tt.simpleDoguName), fmt.Sprintf("UnregisterAllVersions(%v, %v)", testCtx, tt.simpleDoguName))
		})
	}
}

func Test_etcdLocalDoguRegistry_GetCurrent(t *testing.T) {
	tests := []struct {
		name           string
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
		simpleDoguName string
		want           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get current dogu.json",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Get("ldap").Return(nil, assert.AnError)
				return doguReg
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "should succeed to get current dogu.json",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().Get("ldap").Return(testDoguLdap, nil)
				return doguReg
			},
			simpleDoguName: "ldap",
			want:           testDoguLdap,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				doguRegistry: tt.doguRegistryFn(t),
			}
			got, err := er.GetCurrent(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}

func Test_etcdLocalDoguRegistry_GetCurrentOfAll(t *testing.T) {
	tests := []struct {
		name           string
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
		want           []*core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get all current dogu.jsons",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().GetAll().Return(nil, assert.AnError)
				return doguReg
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "should succeed to get all current dogu.jsons",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().GetAll().Return([]*core.Dogu{testDoguLdap, testDoguRedmine, testDoguPostfix}, nil)
				return doguReg
			},
			want:    []*core.Dogu{testDoguLdap, testDoguRedmine, testDoguPostfix},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				doguRegistry: tt.doguRegistryFn(t),
			}
			got, err := er.GetCurrentOfAll(testCtx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCurrentOfAll(%v)", testCtx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCurrentOfAll(%v)", testCtx)
		})
	}
}

func Test_etcdLocalDoguRegistry_IsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		doguRegistryFn func(t *testing.T) registry.DoguRegistry
		simpleDoguName string
		want           bool
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "fail to check if enabled",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().IsEnabled("ldap").Return(false, assert.AnError)
				return doguReg
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i)
			},
		},
		{
			name: "not enabled",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().IsEnabled("ldap").Return(false, nil)
				return doguReg
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr:        assert.NoError,
		},
		{
			name: "enabled",
			doguRegistryFn: func(t *testing.T) registry.DoguRegistry {
				doguReg := extMocks.NewDoguRegistry(t)
				doguReg.EXPECT().IsEnabled("ldap").Return(true, nil)
				return doguReg
			},
			simpleDoguName: "ldap",
			want:           true,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &etcdLocalDoguRegistry{
				doguRegistry: tt.doguRegistryFn(t),
			}
			got, err := er.IsEnabled(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}
