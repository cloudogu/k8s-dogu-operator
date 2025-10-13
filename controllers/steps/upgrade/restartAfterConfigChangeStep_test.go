package upgrade

import (
	"testing"
	"time"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRestartDoguStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguConfigRepo := newMockDoguConfigRepository(t)
		globalConfigRepoMock := newMockGlobalConfigRepository(t)
		doguInterfaceMock := newMockDoguInterface(t)

		step := NewRestartAfterConfigChangeStep(
			doguConfigRepo,
			doguConfigRepo,
			newMockDoguRestartManager(t),
			newMockDeploymentManager(t),
			globalConfigRepoMock,
			doguInterfaceMock,
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
		globalConfigRepositoryFn   func(t *testing.T) globalConfigRepository
		doguInterfaceFn            func(t *testing.T) doguInterface
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository { return newMockGlobalConfigRepository(t) },
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository { return newMockGlobalConfigRepository(t) },
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository { return newMockGlobalConfigRepository(t) },
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository {
					mck := newMockGlobalConfigRepository(t)
					mck.EXPECT().Get(testCtx).Return(config.GlobalConfig{}, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository {
					mck := newMockGlobalConfigRepository(t)
					mck.EXPECT().Get(testCtx).Return(config.GlobalConfig{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository {
					mck := newMockGlobalConfigRepository(t)
					mck.EXPECT().Get(testCtx).Return(config.GlobalConfig{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(
						testCtx,
						&v2.Dogu{
							ObjectMeta: v1.ObjectMeta{
								Namespace: namespace, Name: "test",
							},
						},
						mock.Anything,
						v1.UpdateOptions{}).Return(&v2.Dogu{}, nil)
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
		{
			name: "should fail update status of dogu",
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
				globalConfigRepositoryFn: func(t *testing.T) globalConfigRepository {
					mck := newMockGlobalConfigRepository(t)
					mck.EXPECT().Get(testCtx).Return(config.GlobalConfig{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatusWithRetry(
						testCtx,
						&v2.Dogu{
							ObjectMeta: v1.ObjectMeta{
								Namespace: namespace, Name: "test",
							},
						},
						mock.Anything,
						v1.UpdateOptions{}).Return(nil, assert.AnError)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rds := &RestartAfterConfigChangeStep{
				doguConfigRepository:    tt.fields.doguConfigRepositoryFn(t),
				sensitiveDoguRepository: tt.fields.sensitiveDoguRepositoryFn(t),
				doguRestartManager:      tt.fields.doguRestartManagerFn(t),
				deploymentManager:       tt.fields.deploymentManagerManagerFn(t),
				globalConfigRepository:  tt.fields.globalConfigRepositoryFn(t),
				doguInterface:           tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, rds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
