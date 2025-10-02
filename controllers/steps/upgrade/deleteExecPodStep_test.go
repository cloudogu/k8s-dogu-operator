package upgrade

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDeleteExecPodStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeleteExecPodStep(nil, nil)

		assert.NotNil(t, step)
	})
}

func TestDeleteExecPodStep_Run(t *testing.T) {
	type fields struct {
		execPodFactoryFn   func(t *testing.T) execPodFactory
		localDoguFetcherFn func(t *testing.T) LocalDoguFetcher
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
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("dogu not found in local registry: %w", assert.AnError)),
		},
		{
			name: "should fail to delete exec pod",
			fields: fields{
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Delete(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, &core.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(fmt.Errorf("failed to delete exec pod for dogu %q: %w", "test", assert.AnError)),
		},
		{
			name: "should succeed to delete exec pod",
			fields: fields{
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Delete(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}}, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) LocalDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &DeleteExecPodStep{
				execPodFactory:   tt.fields.execPodFactoryFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, deps.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
