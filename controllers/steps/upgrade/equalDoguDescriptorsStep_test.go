package upgrade

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewEqualDoguDescriptorsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewDeleteExecPodStep(
			&util.ManagerSet{
				LocalDoguFetcher:    newMockLocalDoguFetcher(t),
				ResourceDoguFetcher: newMockResourceDoguFetcher(t),
			},
		)

		assert.NotNil(t, step)
	})
}

func TestEqualDoguDescriptorsStep_Run(t *testing.T) {
	type fields struct {
		resourceDoguFetcherFn func(t *testing.T) resourceDoguFetcher
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
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
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to fetch local dogu descriptor",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should continue if versions are the same",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(&core.Dogu{Version: "1.0.0"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.Continue(),
		},
		{
			name: "should abort if dogu is downgrade",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{Version: "1.0.0"}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(&core.Dogu{Version: "1.0.1"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.Abort(),
		},
		{
			name: "should fail identity check because of different names",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{Name: "test2", Version: "1.0.1"}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(&core.Dogu{Name: "test1", Version: "1.0.0"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.RequeueWithError(fmt.Errorf("dogus must have the same name (%s=%s)", "test1", "test2")),
		},
		{
			name: "should fail identity check because of different namespaces",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{Name: "official/test", Version: "1.0.1"}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(&core.Dogu{Name: "official2/test", Version: "1.0.0"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.RequeueWithError(fmt.Errorf("dogus must have the same namespace (%s=%s)", "official2", "official")),
		},
		{
			name: "should succeed identity check",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}}).Return(&core.Dogu{Name: "official/test", Version: "1.0.1"}, nil, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName(name)).Return(&core.Dogu{Name: "official/test", Version: "1.0.0"}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: name}},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edds := &EqualDoguDescriptorsStep{
				resourceDoguFetcher: tt.fields.resourceDoguFetcherFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, edds.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
