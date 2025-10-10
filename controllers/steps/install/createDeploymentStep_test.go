package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

const doguName = "test"

func TestNewDeploymentStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewCreateDeploymentStep(
			newMockK8sClient(t),
			nil, nil)

		assert.NotNil(t, step)
	})
}

func TestDeploymentStep_Run(t *testing.T) {
	type fields struct {
		upserterFn         func(t *testing.T) resourceUpserter
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		clientFn           func(t *testing.T) k8sClient
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
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: doguName}, &appsv1.Deployment{}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get deployment for dogu %s: %w", doguName, assert.AnError)),
		},
		{
			name: "should fail to fetch local dogu",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(doguName)).Return(nil, assert.AnError)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: doguName}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{Group: v1.SchemeGroupVersion.Group, Resource: doguName}, ""))
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should return if deployment already exists",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: doguName}, &appsv1.Deployment{}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(
						testCtx,
						&v2.Dogu{
							ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
						},
						&cesappcore.Dogu{},
						mock.Anything,
					).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(doguName)).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: doguName}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "successfully upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(
						testCtx,
						&v2.Dogu{
							ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
						},
						&cesappcore.Dogu{},
						mock.Anything,
					).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(doguName)).Return(&cesappcore.Dogu{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Get(testCtx, types.NamespacedName{Namespace: namespace, Name: doguName}, &appsv1.Deployment{}).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: doguName, Namespace: namespace},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &CreateDeploymentStep{
				upserter:         tt.fields.upserterFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				client:           tt.fields.clientFn(t),
			}
			assert.Equalf(t, tt.want, ds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
