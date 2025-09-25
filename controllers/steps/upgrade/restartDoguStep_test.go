package upgrade

import (
	"testing"
	"time"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRestartDoguStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguConfigRepo := newMockDoguConfigRepository(t)
		configMapInterfaceMock := newMockConfigMapInterface(t)

		step := NewRestartDoguStep(
			doguConfigRepo,
			doguConfigRepo,
			newMockDoguRestartManager(t),
			newMockDeploymentManager(t),
			configMapInterfaceMock,
		)

		assert.NotNil(t, step)
	})
}

func TestRestartDoguStep_Run(t *testing.T) {
	type fields struct {
		doguConfigRepositoryFn     func(t *testing.T) doguConfigRepository
		sensitiveDoguRepositoryFn  func(t *testing.T) doguConfigRepository
		doguRestartManagerFn       func(t *testing.T) doguRestartManager
		deploymentManagerManagerFn func(t *testing.T) deploymentManager
		configMapInterfaceFn       func(t *testing.T) configMapInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get last starting time",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(nil, assert.AnError)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					return newMockConfigMapInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to get sensitive config",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{}, assert.AnError)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					return newMockConfigMapInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to get dogu config",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{}, assert.AnError)
					return mck
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{}, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					return newMockConfigMapInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to get global config",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{}, nil)
					return mck
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{}, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, "global-config", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to restart pod of dogu",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{
						Config: config.Config{
							LastUpdated: &v1.Time{
								Time: timestamp,
							},
						},
					}, nil)
					return mck
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{
						Config: config.Config{
							LastUpdated: &v1.Time{
								Time: timestamp,
							},
						},
					}, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					mck := newMockDoguRestartManager(t)
					mck.EXPECT().RestartDogu(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace, Name: "test",
						},
					}).Return(assert.AnError)
					return mck
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, "global-config", v1.GetOptions{}).Return(&v3.ConfigMap{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to restart pod of dogu",
			fields: fields{
				deploymentManagerManagerFn: func(t *testing.T) deploymentManager {
					mck := newMockDeploymentManager(t)
					mck.EXPECT().GetLastStartingTime(testCtx, "test").Return(&time.Time{}, nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{
						Config: config.Config{
							LastUpdated: &v1.Time{
								Time: timestamp,
							},
						},
					}, nil)
					return mck
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockDoguConfigRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleName("test")).Return(config.DoguConfig{
						Config: config.Config{
							LastUpdated: &v1.Time{
								Time: timestamp,
							},
						},
					}, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					mck := newMockDoguRestartManager(t)
					mck.EXPECT().RestartDogu(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Namespace: namespace, Name: "test",
						},
					}).Return(nil)
					return mck
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					mck.EXPECT().Get(testCtx, "global-config", v1.GetOptions{}).Return(&v3.ConfigMap{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rds := &RestartDoguStep{
				doguConfigRepository:    tt.fields.doguConfigRepositoryFn(t),
				sensitiveDoguRepository: tt.fields.sensitiveDoguRepositoryFn(t),
				doguRestartManager:      tt.fields.doguRestartManagerFn(t),
				deploymentManager:       tt.fields.deploymentManagerManagerFn(t),
				configMapInterface:      tt.fields.configMapInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, rds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
