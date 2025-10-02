package upgrade

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeploymentUpdaterStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		upserter := newMockResourceUpserter(t)
		fetcher := newMockLocalDoguFetcher(t)
		step := NewDeploymentUpdaterStep(
			upserter,
			fetcher,
		)

		assert.NotNil(t, step)
		assert.Equal(t, upserter, step.upserter)
		assert.Equal(t, fetcher, step.localDoguFetcher)
	})
}

func TestDeploymentUpdaterStep_Run(t *testing.T) {
	doguUpgradeResource := &v2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v2.DoguSpec{
			Version: "1.0.1",
		},
		Status: v2.DoguStatus{
			InstalledVersion: "1.0.0",
		},
	}

	doguNoUpgradeResource := &v2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: v2.DoguSpec{
			Version: "1.0.0",
		},
		Status: v2.DoguStatus{
			InstalledVersion: "1.0.0",
		},
	}

	type fields struct {
		upserterFn         func(t *testing.T) ResourceUpserter
		localDoguFetcherFn func(t *testing.T) LocalDoguFetcher
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should do nothing on upgrade (deployment should already be updated earlier)",
			fields: fields{
				upserterFn: func(t *testing.T) ResourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
			},
			doguResource: doguUpgradeResource,
			want:         steps.Continue(),
		},
		{
			name: "should requeue on dogu fetch error",
			fields: fields{
				upserterFn: func(t *testing.T) ResourceUpserter {
					return newMockResourceUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) ResourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(testCtx, doguNoUpgradeResource, &core.Dogu{}, mock.Anything).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(&core.Dogu{}, nil)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to upsert deployment",
			fields: fields{
				upserterFn: func(t *testing.T) ResourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguDeployment(testCtx, doguNoUpgradeResource, &core.Dogu{}, mock.Anything).Return(nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, doguNoUpgradeResource).Return(&core.Dogu{}, nil)
					return mck
				},
			},
			doguResource: doguNoUpgradeResource,
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dus := &DeploymentUpdaterStep{
				upserter:         tt.fields.upserterFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, dus.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
