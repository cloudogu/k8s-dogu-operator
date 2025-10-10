package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func TestExecPodCreateStep_Run(t *testing.T) {
	type fields struct {
		clientFn           func(t *testing.T) k8sClient
		recorderFn         func(t *testing.T) eventRecorder
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		execPodFactoryFn   func(t *testing.T) execPodFactory
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get deployment",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &appsv1.Deployment{}).Return(assert.AnError)
					return mck
				},
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get deployment for dogu %s: %w", "test", assert.AnError)),
		},
		{
			name: "should continue on stopped true",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &appsv1.Deployment{}).Return(nil)
					return mck
				},
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec:       v2.DoguSpec{Stopped: true},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}).Return(nil, assert.AnError)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("dogu not found in local registry: %w", assert.AnError)),
		},
		{
			name: "should fail to create exec pod",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
				recorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Eventf(&v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}, coreV1.EventTypeNormal, InstallEventReason, "Starting execPod...")
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Create(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}, &core.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to create execPod for dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should succeed to create exec pod",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Name: "test", Namespace: namespace}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
				recorderFn: func(t *testing.T) eventRecorder {
					mck := newMockEventRecorder(t)
					mck.EXPECT().Eventf(&v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}, coreV1.EventTypeNormal, InstallEventReason, "Starting execPod...")
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Create(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
					}, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test", Namespace: namespace},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			epcs := &CreateExecPodStep{
				client:           tt.fields.clientFn(t),
				recorder:         tt.fields.recorderFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				execPodFactory:   tt.fields.execPodFactoryFn(t),
			}
			assert.Equalf(t, tt.want, epcs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
