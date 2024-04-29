package localregistry

import (
	"context"
	"fmt"
	"testing"

	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"

	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

var testCtx = context.Background()
var testDogu = &core.Dogu{
	Name:        "official/ldap",
	Version:     "1.2.3-4",
	DisplayName: "ldap",
	Description: "some description",
	Image:       "registry.cloudogu.com/official/ldap:1.2.3-4",
}

func TestNewCombinedLocalDoguRegistry(t *testing.T) {
	// given
	doguClientMock := mocks.NewDoguInterface(t)
	cmClientMock := extMocks.NewConfigMapInterface(t)

	etcdDoguRegMock := extMocks.NewDoguRegistry(t)
	etcdRegMock := extMocks.NewConfigurationRegistry(t)
	etcdRegMock.EXPECT().DoguRegistry().Return(etcdDoguRegMock)

	// when
	localReg := NewCombinedLocalDoguRegistry(doguClientMock, cmClientMock, etcdRegMock)

	// then
	assert.NotEmpty(t, localReg)
	assert.IsType(t, &clusterNativeLocalDoguRegistry{}, localReg.cnRegistry)
	assert.IsType(t, &etcdLocalDoguRegistry{}, localReg.etcdRegistry)
}

func TestCombinedLocalDoguRegistry_Enable(t *testing.T) {
	tests := []struct {
		name           string
		cnRegistryFn   func(t *testing.T) LocalDoguRegistry
		etcdRegistryFn func(t *testing.T) LocalDoguRegistry
		dogu           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail in cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Enable(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Enable(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to enable dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should fail in etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Enable(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Enable(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to enable dogu \"official/ldap\" in ETCD local registry (legacy)", i)
			},
		},
		{
			name: "should fail in cluster-native and etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Enable(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Enable(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to enable dogu \"official/ldap\" in ETCD local registry (legacy)", i) &&
					assert.ErrorContains(t, err, "failed to enable dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should succeed",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Enable(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Enable(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu:    testDogu,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CombinedLocalDoguRegistry{
				cnRegistry:   tt.cnRegistryFn(t),
				etcdRegistry: tt.etcdRegistryFn(t),
			}
			tt.wantErr(t, cr.Enable(testCtx, tt.dogu), fmt.Sprintf("Enable(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func TestCombinedLocalDoguRegistry_Register(t *testing.T) {
	tests := []struct {
		name           string
		cnRegistryFn   func(t *testing.T) LocalDoguRegistry
		etcdRegistryFn func(t *testing.T) LocalDoguRegistry
		dogu           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail in cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Register(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Register(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to register dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should fail in etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Register(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Register(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to register dogu \"official/ldap\" in ETCD local registry (legacy)", i)
			},
		},
		{
			name: "should fail in cluster-native and etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Register(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Register(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to register dogu \"official/ldap\" in ETCD local registry (legacy)", i) &&
					assert.ErrorContains(t, err, "failed to register dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should succeed",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Register(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Register(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu:    testDogu,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CombinedLocalDoguRegistry{
				cnRegistry:   tt.cnRegistryFn(t),
				etcdRegistry: tt.etcdRegistryFn(t),
			}
			tt.wantErr(t, cr.Register(testCtx, tt.dogu), fmt.Sprintf("Register(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func TestCombinedLocalDoguRegistry_Reregister(t *testing.T) {
	tests := []struct {
		name           string
		cnRegistryFn   func(t *testing.T) LocalDoguRegistry
		etcdRegistryFn func(t *testing.T) LocalDoguRegistry
		dogu           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail in cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Reregister(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Reregister(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to reregister dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should fail in etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Reregister(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Reregister(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to reregister dogu \"official/ldap\" in ETCD local registry (legacy)", i)
			},
		},
		{
			name: "should fail in cluster-native and etcd registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Reregister(testCtx, testDogu).Return(assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Reregister(testCtx, testDogu).Return(assert.AnError)
				return etcdRegMock
			},
			dogu: testDogu,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to reregister dogu \"official/ldap\" in ETCD local registry (legacy)", i) &&
					assert.ErrorContains(t, err, "failed to reregister dogu \"official/ldap\" in cluster-native local registry", i)
			},
		},
		{
			name: "should succeed",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().Reregister(testCtx, testDogu).Return(nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().Reregister(testCtx, testDogu).Return(nil)
				return etcdRegMock
			},
			dogu:    testDogu,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CombinedLocalDoguRegistry{
				cnRegistry:   tt.cnRegistryFn(t),
				etcdRegistry: tt.etcdRegistryFn(t),
			}
			tt.wantErr(t, cr.Reregister(testCtx, tt.dogu), fmt.Sprintf("Reregister(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func TestCombinedLocalDoguRegistry_GetCurrent(t *testing.T) {
	tests := []struct {
		name           string
		cnRegistryFn   func(t *testing.T) LocalDoguRegistry
		etcdRegistryFn func(t *testing.T) LocalDoguRegistry
		simpleDoguName string
		want           *core.Dogu
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get current dogu.json from cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(nil, assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get current dogu.json of \"ldap\" from cluster-native local registry", i)
			},
		},
		{
			name: "should not find current dogu.json in cluster-native registry and fail for etcd",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(nil, k8sErrs.NewNotFound(schema.GroupResource{}, getConfigMapName(testDogu)))
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(nil, assert.AnError)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get current dogu.json of \"ldap\" from ETCD local registry (legacy/fallback)", i)
			},
		},
		{
			name: "should not find current dogu.json in cluster-native registry and succeed in ETCD registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(nil, k8sErrs.NewNotFound(schema.GroupResource{}, getConfigMapName(testDogu)))
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(testDogu, nil)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           testDogu,
			wantErr:        assert.NoError,
		},
		{
			name: "should get current dogu.json from cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().GetCurrent(testCtx, "ldap").Return(testDogu, nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           testDogu,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CombinedLocalDoguRegistry{
				cnRegistry:   tt.cnRegistryFn(t),
				etcdRegistry: tt.etcdRegistryFn(t),
			}
			got, err := cr.GetCurrent(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}

func TestCombinedLocalDoguRegistry_IsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		cnRegistryFn   func(t *testing.T) LocalDoguRegistry
		etcdRegistryFn func(t *testing.T) LocalDoguRegistry
		simpleDoguName string
		want           bool
		wantErr        assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to check if dogu is enabled in cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, assert.AnError)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to check if dogu \"ldap\" is enabled in cluster-native local registry", i)
			},
		},
		{
			name: "dogu is not enabled in cluster-native registry; fail to check in ETCD registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, assert.AnError)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to check if dogu \"ldap\" is enabled in ETCD local registry (legacy/fallback)", i)
			},
		},
		{
			name: "dogu is not enabled in cluster-native registry or ETCD registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr:        assert.NoError,
		},
		{
			name: "dogu is not enabled in cluster-native registry or but in ETCD registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(false, nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				etcdRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(true, nil)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           true,
			wantErr:        assert.NoError,
		},
		{
			name: "dogu is enabled in cluster-native registry",
			cnRegistryFn: func(t *testing.T) LocalDoguRegistry {
				cnRegMock := mocks.NewLocalDoguRegistry(t)
				cnRegMock.EXPECT().IsEnabled(testCtx, "ldap").Return(true, nil)
				return cnRegMock
			},
			etcdRegistryFn: func(t *testing.T) LocalDoguRegistry {
				etcdRegMock := mocks.NewLocalDoguRegistry(t)
				return etcdRegMock
			},
			simpleDoguName: "ldap",
			want:           true,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := &CombinedLocalDoguRegistry{
				cnRegistry:   tt.cnRegistryFn(t),
				etcdRegistry: tt.etcdRegistryFn(t),
			}
			got, err := cr.IsEnabled(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}
