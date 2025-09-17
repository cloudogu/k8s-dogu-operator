package upgrade

import (
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeploymentUpdaterStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeploymentUpdaterStep(
			nil,
			newMockLocalDoguFetcher(t),
			newMockDeploymentInterface(t),
		)

		assert.NotNil(t, step)
	})
}

func TestDeploymentUpdaterStep_Run(t *testing.T) {
	type fields struct {
		upserterFn            func(t *testing.T) resourceUpserter
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		deploymentInterfaceFn func(t *testing.T) deploymentInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch dogu deployment",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to fetch dogu resource",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(&appsv1.Deployment{}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(testCtx, &v2.Dogu{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
					},
						&core.Dogu{},
						mock.Anything,
					).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(testCtx, &v2.Dogu{
						ObjectMeta: metav1.ObjectMeta{Name: "test"},
					},
						&core.Dogu{},
						mock.Anything,
					).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				deploymentInterfaceFn: func(t *testing.T) deploymentInterface {
					mck := newMockDeploymentInterface(t)
					mck.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dus := &DeploymentUpdaterStep{
				upserter:            tt.fields.upserterFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				deploymentInterface: tt.fields.deploymentInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, dus.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
