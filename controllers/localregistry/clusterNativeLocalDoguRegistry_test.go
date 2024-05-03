package localregistry

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

var ldapDoguJson = "{\"Name\":\"official/ldap\",\"Version\":\"1.2.3-4\",\"DisplayName\":\"ldap\",\"Description\":\"some description\",\"Category\":\"\",\"Tags\":null,\"Logo\":\"\",\"URL\":\"\",\"Image\":\"registry.cloudogu.com/official/ldap:1.2.3-4\",\"ExposedPorts\":null,\"ExposedCommands\":null,\"Volumes\":null,\"HealthCheck\":{\"Type\":\"\",\"State\":\"\",\"Port\":0,\"Path\":\"\",\"Parameters\":null},\"HealthChecks\":null,\"ServiceAccounts\":null,\"Privileged\":false,\"Configuration\":null,\"Properties\":null,\"EnvironmentVariables\":null,\"Dependencies\":null,\"OptionalDependencies\":null}"
var postfixDoguJson = "{\"Name\":\"official/postfix\",\"Version\":\"1.2.3-4\",\"DisplayName\":\"postfix\",\"Description\":\"some description\",\"Category\":\"\",\"Tags\":null,\"Logo\":\"\",\"URL\":\"\",\"Image\":\"registry.cloudogu.com/official/postfix:1.2.3-4\",\"ExposedPorts\":null,\"ExposedCommands\":null,\"Volumes\":null,\"HealthCheck\":{\"Type\":\"\",\"State\":\"\",\"Port\":0,\"Path\":\"\",\"Parameters\":null},\"HealthChecks\":null,\"ServiceAccounts\":null,\"Privileged\":false,\"Configuration\":null,\"Properties\":null,\"EnvironmentVariables\":null,\"Dependencies\":null,\"OptionalDependencies\":null}"

func Test_clusterNativeLocalDoguRegistry_Enable(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		dogu              *core.Dogu
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get configmap",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, assert.AnError)
				return cmClient
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail to update configmap",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: make(map[string]string)}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				cmClient.EXPECT().Update(testCtx, configMap, metav1.UpdateOptions{}).Run(func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) {
					assert.Equal(t, testDoguLdap.Version, configMap.Data[currentVersionKey])
				}).Return(configMap, assert.AnError)
				return cmClient
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to update local registry for dogu \"ldap\" with new version", i)
			},
		},
		{
			name: "should succeed to update configmap",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: make(map[string]string)}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				cmClient.EXPECT().Update(testCtx, configMap, metav1.UpdateOptions{}).Run(func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) {
					assert.Equal(t, testDoguLdap.Version, configMap.Data[currentVersionKey])
				}).Return(configMap, nil)
				return cmClient
			},
			dogu:    testDoguLdap,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			tt.wantErr(t, cmr.Enable(testCtx, tt.dogu), fmt.Sprintf("Enable(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func Test_clusterNativeLocalDoguRegistry_Register(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		dogu              *core.Dogu
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to serialize dogu.json and get configmap",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, assert.AnError)
				return cmClient
			},
			dogu: &core.Dogu{
				Name:    "official/ldap",
				Volumes: []core.Volume{{Clients: []core.VolumeClient{{Params: make(map[interface{}]interface{})}}}},
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to serialize dogu.json of \"official/ldap\"", i) &&
					assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should succeed to create configmap if not found",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, k8sErrs.NewNotFound(schema.GroupResource{}, "dogu-spec-ldap"))
				expectedCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dogu-spec-ldap",
						Labels: map[string]string{
							appLabelKey:      appLabelValueCes,
							doguNameLabelKey: "ldap",
							typeLabelKey:     typeLabelValueLocalDoguRegistry,
						},
					},
					Data: map[string]string{"1.2.3-4": ldapDoguJson},
				}
				cmClient.EXPECT().Create(testCtx, expectedCm, metav1.CreateOptions{}).Return(expectedCm, nil)
				return cmClient
			},
			dogu:    testDoguLdap,
			wantErr: assert.NoError,
		},
		{
			name: "should fail to create configmap if not found",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, k8sErrs.NewNotFound(schema.GroupResource{}, "dogu-spec-ldap"))
				expectedCm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dogu-spec-ldap",
						Labels: map[string]string{
							appLabelKey:      appLabelValueCes,
							doguNameLabelKey: "ldap",
							typeLabelKey:     typeLabelValueLocalDoguRegistry,
						},
					},
					Data: map[string]string{"1.2.3-4": ldapDoguJson},
				}
				cmClient.EXPECT().Create(testCtx, expectedCm, metav1.CreateOptions{}).Return(nil, assert.AnError)
				return cmClient
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to create local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail to update existing configmap with new version",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dogu-spec-ldap",
						Labels: map[string]string{
							appLabelKey:      appLabelValueCes,
							doguNameLabelKey: "ldap",
							typeLabelKey:     typeLabelValueLocalDoguRegistry,
						},
					},
					Data: make(map[string]string),
				}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				cmClient.EXPECT().Update(testCtx, configMap, metav1.UpdateOptions{}).Run(func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) {
					assert.Contains(t, configMap.Data, testDoguLdap.Version)
					assert.Equal(t, configMap.Data[testDoguLdap.Version], ldapDoguJson)
				}).Return(nil, assert.AnError)
				return cmClient
			},
			dogu: testDoguLdap,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to add local registry entry for dogu \"official/ldap\"", i)
			},
		},
		{
			name: "should succeed to update existing configmap with new version",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "dogu-spec-ldap",
						Labels: map[string]string{
							appLabelKey:      appLabelValueCes,
							doguNameLabelKey: "ldap",
							typeLabelKey:     typeLabelValueLocalDoguRegistry,
						},
					},
					Data: make(map[string]string),
				}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				cmClient.EXPECT().Update(testCtx, configMap, metav1.UpdateOptions{}).Run(func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) {
					assert.Contains(t, configMap.Data, testDoguLdap.Version)
					assert.Equal(t, configMap.Data[testDoguLdap.Version], ldapDoguJson)
				}).Return(configMap, nil)
				return cmClient
			},
			dogu:    testDoguLdap,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			tt.wantErr(t, cmr.Register(testCtx, tt.dogu), fmt.Sprintf("Register(%v, %v)", testCtx, tt.dogu))
		})
	}
}

func Test_clusterNativeLocalDoguRegistry_UnregisterAllVersions(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		simpleDoguName    string
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to delete config map",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Delete(testCtx, "dogu-spec-ldap", metav1.DeleteOptions{}).Return(assert.AnError)
				return cmClient
			},
			simpleDoguName: "ldap",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to delete local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should succeed if config map is not found",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Delete(testCtx, "dogu-spec-ldap", metav1.DeleteOptions{}).Return(k8sErrs.NewNotFound(schema.GroupResource{}, "dogu-spec-ldap"))
				return cmClient
			},
			simpleDoguName: "ldap",
			wantErr:        assert.NoError,
		},
		{
			name: "should succeed to delete config map",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Delete(testCtx, "dogu-spec-ldap", metav1.DeleteOptions{}).Return(nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			tt.wantErr(t, cmr.UnregisterAllVersions(testCtx, tt.simpleDoguName), fmt.Sprintf("UnregisterAllVersions(%v, %v)", testCtx, tt.simpleDoguName))
		})
	}
}

func Test_clusterNativeLocalDoguRegistry_GetCurrent(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		simpleDoguName    string
		want              *core.Dogu
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get spec config map",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, assert.AnError)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail because current version key does not exist",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: make(map[string]string)}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "dogu registry does not contain currently installed version for dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail because dogu.json for current version does not exist",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: map[string]string{"current": "1.2.3-4", "1.2.3-3": "this is not 1.2.3-4"}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "dogu registry does not contain dogu.json for currently installed version of dogu \"ldap\"", i)
			},
		},
		{
			name: "should fail to parse dogu.json",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: map[string]string{"current": "1.2.3-4", "1.2.3-4": "{/{{unparsable}"}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to parse current dogu.json of \"ldap\"", i)
			},
		},
		{
			name: "should succeed",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: map[string]string{"current": "1.2.3-4", "1.2.3-4": ldapDoguJson}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           testDoguLdap,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			got, err := cmr.GetCurrent(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCurrent(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}

func Test_clusterNativeLocalDoguRegistry_GetCurrentOfAll(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		want              []*core.Dogu
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to list config maps",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s,%s=%s", appLabelKey, appLabelValueCes, doguNameLabelKey, typeLabelKey, typeLabelValueLocalDoguRegistry)}).Return(nil, assert.AnError)
				return cmClient
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to list cluster native local dogu registries", i)
			},
		},
		{
			name: "should fail to parse multiple dogu.jsons",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMaps := &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "dogu-spec-ldap",
							Labels: map[string]string{
								appLabelKey:      appLabelValueCes,
								doguNameLabelKey: "ldap",
								typeLabelKey:     typeLabelValueLocalDoguRegistry,
							},
						},
						Data: map[string]string{
							currentVersionKey: "1.2.3-4",
							"1.2.3-4":         "{/{{unparsable}",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "dogu-spec-postfix",
							Labels: map[string]string{
								appLabelKey:      appLabelValueCes,
								doguNameLabelKey: "postfix",
								typeLabelKey:     typeLabelValueLocalDoguRegistry,
							},
						},
						Data: map[string]string{
							currentVersionKey: "1.2.3-4",
							"1.2.3-4":         "{/{{unparsable}",
						},
					},
				}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s,%s=%s", appLabelKey, appLabelValueCes, doguNameLabelKey, typeLabelKey, typeLabelValueLocalDoguRegistry)}).Return(configMaps, nil)
				return cmClient
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to get some dogu specs", i) &&
					assert.ErrorContains(t, err, "failed to parse current dogu.json of \"ldap\"", i) &&
					assert.ErrorContains(t, err, "failed to parse current dogu.json of \"postfix\"", i)
			},
		},
		{
			name: "should succeed",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMaps := &corev1.ConfigMapList{Items: []corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "dogu-spec-ldap",
							Labels: map[string]string{
								appLabelKey:      appLabelValueCes,
								doguNameLabelKey: "ldap",
								typeLabelKey:     typeLabelValueLocalDoguRegistry,
							},
						},
						Data: map[string]string{
							currentVersionKey: "1.2.3-4",
							"1.2.3-4":         ldapDoguJson,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "dogu-spec-postfix",
							Labels: map[string]string{
								appLabelKey:      appLabelValueCes,
								doguNameLabelKey: "postfix",
								typeLabelKey:     typeLabelValueLocalDoguRegistry,
							},
						},
						Data: map[string]string{
							currentVersionKey: "1.2.3-4",
							"1.2.3-4":         postfixDoguJson,
						},
					},
				}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().List(testCtx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s,%s,%s=%s", appLabelKey, appLabelValueCes, doguNameLabelKey, typeLabelKey, typeLabelValueLocalDoguRegistry)}).Return(configMaps, nil)
				return cmClient
			},
			want:    []*core.Dogu{testDoguLdap, testDoguPostfix},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			got, err := cmr.GetCurrentOfAll(testCtx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetCurrentOfAll(%v)", testCtx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetCurrentOfAll(%v)", testCtx)
		})
	}
}

func Test_clusterNativeLocalDoguRegistry_IsEnabled(t *testing.T) {
	tests := []struct {
		name              string
		configMapClientFn func(t *testing.T) v1.ConfigMapInterface
		simpleDoguName    string
		want              bool
		wantErr           assert.ErrorAssertionFunc
	}{
		{
			name: "should fail to get config map",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(nil, assert.AnError)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorIs(t, err, assert.AnError, i) &&
					assert.ErrorContains(t, err, "failed to get local registry for dogu \"ldap\"", i)
			},
		},
		{
			name: "should return false",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: map[string]string{}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           false,
			wantErr:        assert.NoError,
		},
		{
			name: "should return true",
			configMapClientFn: func(t *testing.T) v1.ConfigMapInterface {
				configMap := &corev1.ConfigMap{Data: map[string]string{"current": "1.2.3-4"}}
				cmClient := extMocks.NewConfigMapInterface(t)
				cmClient.EXPECT().Get(testCtx, "dogu-spec-ldap", metav1.GetOptions{}).Return(configMap, nil)
				return cmClient
			},
			simpleDoguName: "ldap",
			want:           true,
			wantErr:        assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmr := &clusterNativeLocalDoguRegistry{
				configMapClient: tt.configMapClientFn(t),
			}
			got, err := cmr.IsEnabled(testCtx, tt.simpleDoguName)
			if !tt.wantErr(t, err, fmt.Sprintf("IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsEnabled(%v, %v)", testCtx, tt.simpleDoguName)
		})
	}
}
