package postinstall

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v3 "k8s.io/api/apps/v1"
	v4 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewRestartDoguStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		podInterfaceMock := newMockPodInterface(t)
		coreV1InterfaceMock := newMockCoreV1Interface(t)
		coreV1InterfaceMock.EXPECT().Pods(namespace).Return(podInterfaceMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1InterfaceMock)

		step := NewRestartDoguStep(
			newMockK8sClient(t),
			&util.ManagerSet{
				ClientSet:        clientSetMock,
				LocalDoguFetcher: newMockLocalDoguFetcher(t),
				ResourceUpserter: newMockResourceUpserter(t),
			},
			namespace,
			util.ConfigRepositories{
				DoguConfigRepository:    &repository.DoguConfigRepository{},
				SensitiveDoguRepository: &repository.DoguConfigRepository{},
			},
			newMockDoguRestartManager(t),
		)

		assert.NotNil(t, step)
	})
}

func TestRestartDoguStep_Run(t *testing.T) {
	type fields struct {
		clientFn                  func(t *testing.T) k8sClient
		doguConfigRepositoryFn    func(t *testing.T) doguConfigRepository
		sensitiveDoguRepositoryFn func(t *testing.T) doguConfigRepository
		podInterfaceFn            func(t *testing.T) podInterface
		doguRestartManagerFn      func(t *testing.T) doguRestartManager
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get deployment of dogu",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(assert.AnError)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace, Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get deployment for dogu %s: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to list pods of deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(nil)
					return mck
				},
				doguConfigRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				sensitiveDoguRepositoryFn: func(t *testing.T) doguConfigRepository {
					return newMockDoguConfigRepository(t)
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					mck := newMockPodInterface(t)
					mck.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: v1.FormatLabelSelector(nil)}).Return(nil, assert.AnError)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
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
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(nil)
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
				podInterfaceFn: func(t *testing.T) podInterface {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockPodInterface(t)
					pods := &v4.PodList{
						Items: []v4.Pod{
							{
								Status: v4.PodStatus{
									StartTime: &v1.Time{
										Time: timestamp,
									},
								},
							},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: v1.FormatLabelSelector(nil)}).Return(pods, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
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
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(nil)
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
				podInterfaceFn: func(t *testing.T) podInterface {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2025-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockPodInterface(t)
					pods := &v4.PodList{
						Items: []v4.Pod{
							{
								Status: v4.PodStatus{
									StartTime: &v1.Time{
										Time: timestamp,
									},
								},
							},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: v1.FormatLabelSelector(nil)}).Return(pods, nil)
					return mck
				},
				doguRestartManagerFn: func(t *testing.T) doguRestartManager {
					return newMockDoguRestartManager(t)
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
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(nil)
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
				podInterfaceFn: func(t *testing.T) podInterface {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2024-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockPodInterface(t)
					pods := &v4.PodList{
						Items: []v4.Pod{
							{
								Status: v4.PodStatus{
									StartTime: &v1.Time{
										Time: timestamp,
									},
								},
							},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: v1.FormatLabelSelector(nil)}).Return(pods, nil)
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
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &v3.Deployment{}).Return(nil)
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
				podInterfaceFn: func(t *testing.T) podInterface {
					layout := "2006-01-02T15:04:05.000Z"
					str := "2024-11-12T11:45:26.371Z"
					timestamp, err := time.Parse(layout, str)
					require.NoError(t, err)
					mck := newMockPodInterface(t)
					pods := &v4.PodList{
						Items: []v4.Pod{
							{
								Status: v4.PodStatus{
									StartTime: &v1.Time{
										Time: timestamp,
									},
								},
							},
						},
					}
					mck.EXPECT().List(testCtx, v1.ListOptions{LabelSelector: v1.FormatLabelSelector(nil)}).Return(pods, nil)
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
				client:                  tt.fields.clientFn(t),
				doguConfigRepository:    tt.fields.doguConfigRepositoryFn(t),
				sensitiveDoguRepository: tt.fields.sensitiveDoguRepositoryFn(t),
				podInterface:            tt.fields.podInterfaceFn(t),
				doguRestartManager:      tt.fields.doguRestartManagerFn(t),
			}
			assert.Equalf(t, tt.want, rds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
