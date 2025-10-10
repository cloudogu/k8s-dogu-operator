package postinstall

import (
	context "context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v3 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewReplicasStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		deploymentInterfaceMock := newMockDeploymentInterface(t)
		fetcher := newMockLocalDoguFetcher(t)
		doguInterfaceMock := newMockDoguInterface(t)

		step := NewStartStopStep(
			newMockK8sClient(t),
			deploymentInterfaceMock,
			fetcher,
			doguInterfaceMock,
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
			name: "should fail to get data pvc",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					pvc := &corev1.PersistentVolumeClaim{}
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test"}, pvc).Return(assert.AnError)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{
						Volumes: []core.Volume{
							{
								NeedsBackup: true,
							},
						},
					}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to get data pvc for dogu test: %w", assert.AnError)),
		},
		{
			name: "should check if pvc is resized and continue",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					pvc := &corev1.PersistentVolumeClaim{}
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test"}, pvc).Return(nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().GetScale(testCtx, "test", v1.GetOptions{}).Return(&v3.Scale{}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{
						Volumes: []core.Volume{
							{
								NeedsBackup: true,
							},
						},
					}, nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Resources: v2.DoguResources{
						MinDataVolumeSize: *resource.NewQuantity(1, "DecimalSI"),
					},
				},
			},
			want: steps.Continue(),
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
					mck.EXPECT().UpdateStatusWithRetry(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Stopped: false,
						},
					}, mock.Anything, v1.UpdateOptions{}).Return(nil, assert.AnError)
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
					doguCr := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Stopped: false,
						},
					}
					mck.EXPECT().UpdateStatusWithRetry(testCtx, doguCr, mock.Anything, v1.UpdateOptions{}).Return(nil, nil).Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts v1.UpdateOptions) {
						modifyStatusFn(doguCr.Status)
						assert.Equal(t, false, doguCr.Status.Stopped)
					})
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
			rs := &StartStopStep{
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
				client:              tt.fields.clientFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				doguInterface:       tt.fields.doguInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, rs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
