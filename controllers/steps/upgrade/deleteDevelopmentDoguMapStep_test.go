package upgrade

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

const namespace = "ecosystem"
const name = "test"

func TestNewDeleteDevelopmentDoguMapStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeleteDevelopmentDoguMapStep(
			newMockK8sClient(t),
			&util.ManagerSet{
				ResourceDoguFetcher: newMockResourceDoguFetcher(t),
			},
		)

		assert.NotNil(t, step)
	})
}

func TestDeleteDevelopmentDoguMapStep_Run(t *testing.T) {
	type fields struct {
		resourceDoguFetcherFn func(t *testing.T) resourceDoguFetcher
		clientFn              func(t *testing.T) k8sClient
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch remote dogu descriptor",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{}).Return(nil, nil, assert.AnError)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(fmt.Errorf("dogu upgrade failed: %w", assert.AnError)),
		},
		{
			name: "should fail to delete development map from cluster",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
						Spec: v2.DoguSpec{
							Version: "1.0.0",
						},
					}).Return(&core.Dogu{}, &v2.DevelopmentDoguMap{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					ddm := &v2.DevelopmentDoguMap{}
					mck.EXPECT().Delete(testCtx, ddm.ToConfigMap()).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("dogu upgrade %s:%s failed: %w", "test", "1.0.0", fmt.Errorf("failed to delete custom dogu development map : %w", assert.AnError))),
		},
		{
			name: "should succeed to delete development map from cluster",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
						Spec: v2.DoguSpec{
							Version: "1.0.0",
						},
					}).Return(&core.Dogu{}, &v2.DevelopmentDoguMap{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					ddm := &v2.DevelopmentDoguMap{}
					mck.EXPECT().Delete(testCtx, ddm.ToConfigMap()).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ddms := &DeleteDevelopmentDoguMapStep{
				resourceDoguFetcher: tt.fields.resourceDoguFetcherFn(t),
				client:              tt.fields.clientFn(t),
			}
			assert.Equalf(t, tt.want, ddms.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
