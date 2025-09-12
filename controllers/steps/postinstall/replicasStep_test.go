package postinstall

import (
	"testing"
	"time"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v3 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewReplicasStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		appV1InterfaceMock := newMockAppV1Interface(t)
		appV1InterfaceMock.EXPECT().Deployments(namespace).Return(deploymentInterfaceMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appV1InterfaceMock)

		doguInterfaceMock := newMockDoguInterface(t)
		ecosystemInterfaceMock := newMockEcosystemInterface(t)
		ecosystemInterfaceMock.EXPECT().Dogus(namespace).Return(doguInterfaceMock)

		step := NewReplicasStep(
			newMockK8sClient(t),
			&util.ManagerSet{
				ClientSet:        clientSetMock,
				EcosystemClient:  ecosystemInterfaceMock,
				LocalDoguFetcher: newMockLocalDoguFetcher(t),
			},
			namespace,
		)

		assert.NotNil(t, step)
	})
}

func TestReplicasStep_Run(t *testing.T) {
	type fields struct {
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
		clientFn              func(t *testing.T) k8sClient
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		doguInterfaceFn       func(t *testing.T) doguInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get scale of deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{}, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should abort if dogu is stopped",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Stopped: true,
				},
			},
			want: steps.Abort(),
		},
		{
			name: "should continue if dogu is not stopped and running",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 1,
						},
					}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Stopped: false,
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to update scale of deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 0,
						},
					}, nil)
					mck.EXPECT().UpdateScale(testCtx, "test", &v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 1,
						},
					}, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Stopped: false,
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update status",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 0,
						},
					}, nil)
					mck.EXPECT().UpdateScale(testCtx, "test", &v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 1,
						},
					}, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Stopped: false,
						},
					}, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Stopped: false,
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update status",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 0,
						},
					}, nil)
					mck.EXPECT().UpdateScale(testCtx, "test", &v3.Scale{
						Spec: v3.ScaleSpec{
							Replicas: 1,
						},
					}, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Stopped: false,
						},
					}, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Stopped: false,
				},
			},
			want: steps.RequeueAfter(5 * time.Second),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &ReplicasStep{
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				client:              tt.fields.clientFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				doguInterface:       tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, rs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
