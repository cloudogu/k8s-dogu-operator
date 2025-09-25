package postinstall

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestNewVolumeExpanderStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguInterfaceMock := newMockDoguInterface(t)
		fetcher := newMockLocalDoguFetcher(t)

		step := NewVolumeExpanderStep(
			newMockK8sClient(t),
			doguInterfaceMock,
			fetcher,
		)

		assert.NotNil(t, step)
	})
}

func TestVolumeExpanderStep_Run(t *testing.T) {
	type fields struct {
		clientFn           func(t *testing.T) k8sClient
		doguInterfaceFn    func(t *testing.T) doguInterface
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update success condition if no volume exists",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					condition := v1.Condition{
						Type:               v2.ConditionMeetsMinVolumeSize,
						Status:             v1.ConditionTrue,
						Reason:             ActualVolumeSizeMeetsMinDataSize,
						Message:            "Current VolumeSize meets the configured minimum VolumeSize",
						LastTransitionTime: v1.Now().Rfc3339Copy(),
					}
					d := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: v2.DoguStatus{
							Conditions: []v1.Condition{condition},
						},
					}
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, d, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should reconcile if volume is expanded",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					condition := v1.Condition{
						Type:               v2.ConditionMeetsMinVolumeSize,
						Status:             v1.ConditionTrue,
						Reason:             ActualVolumeSizeMeetsMinDataSize,
						Message:            "Current VolumeSize meets the configured minimum VolumeSize",
						LastTransitionTime: v1.Now().Rfc3339Copy(),
					}
					d := &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Status: v2.DoguStatus{
							Conditions: []v1.Condition{condition},
						},
					}
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, d, v1.UpdateOptions{}).Return(d, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
		{
			name: "should fail to get data pvc",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test"}, &corev1.PersistentVolumeClaim{}).Return(assert.AnError)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{
						Name:    "test",
						Volumes: []core.Volume{{Name: "test", NeedsBackup: true}},
					}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to get data pvc for dogu test: %w", assert.AnError)),
		},
		{
			name: "should fail to set success condition if resized",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test"}, &corev1.PersistentVolumeClaim{}).Return(nil)
					mck.EXPECT().Update(testCtx, mock.Anything).Return(nil)
					return mck
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					mck := newMockDoguInterface(t)
					mck.EXPECT().UpdateStatus(testCtx, mock.Anything, v1.UpdateOptions{}).Return(&v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{
						Name:    "test",
						Volumes: []core.Volume{{Name: "test", NeedsBackup: true}},
					}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueAfter(requeueAfterVolume),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &VolumeExpanderStep{
				client:           tt.fields.clientFn(t),
				doguInterface:    tt.fields.doguInterfaceFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, vs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
