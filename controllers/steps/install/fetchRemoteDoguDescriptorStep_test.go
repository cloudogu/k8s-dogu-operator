package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v3 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewFetchRemoteDoguDescriptorStep(t *testing.T) {
	step := NewFetchRemoteDoguDescriptorStep(newMockK8sClient(t), newMockLocalDoguDescriptorRepository(t), newMockResourceDoguFetcher(t))
	assert.NotEmpty(t, step)
}

func TestFetchRemoteDoguDescriptorStep_Run(t *testing.T) {
	type fields struct {
		clientFn                  func(t *testing.T) k8sClient
		resourceDoguFetcherFn     func(t *testing.T) resourceDoguFetcher
		localDoguDescriptorRepoFn func(t *testing.T) localDoguDescriptorRepository
	}
	tests := []struct {
		name     string
		fields   fields
		resource *v2.Dogu
		want     steps.StepResult
	}{
		{
			name: "should fail to get SimpleNameVersion",
			fields: fields{
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					return newMockLocalDoguDescriptorRepository(t)
				},
			},
			resource: &v2.Dogu{
				Spec: v2.DoguSpec{
					Version: "---",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("found more than one hyphen in version ---")),
		},
		{
			name: "should fail to get local dogu descriptor",
			fields: fields{
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					mck := newMockLocalDoguDescriptorRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleNameVersion{Name: "test", Version: core.Version{Raw: "1.0.0", Major: 1}}).Return(nil, assert.AnError)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
			},
			resource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should continue if local dogu descriptor is found",
			fields: fields{
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					mck := newMockLocalDoguDescriptorRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleNameVersion{Name: "test", Version: core.Version{Raw: "1.0.0", Major: 1}}).Return(&core.Dogu{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
			},
			resource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail to fetch remote dogu descriptor",
			fields: fields{
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					mck := newMockLocalDoguDescriptorRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleNameVersion{Name: "test", Version: core.Version{Raw: "1.0.0", Major: 1}}).Return(nil, errors.NewNotFoundError(assert.AnError))
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Version: "1.0.0",
						},
					}).Return(nil, nil, assert.AnError)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
			},
			resource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to add remote dogu descriptor to local dogu descriptor repo",
			fields: fields{
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					mck := newMockLocalDoguDescriptorRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleNameVersion{Name: "test", Version: core.Version{Raw: "1.0.0", Major: 1}}).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().Add(testCtx, dogu.SimpleName("test"), &core.Dogu{Name: "test", Version: "1.0.0"}).Return(assert.AnError)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Version: "1.0.0",
						},
					}).Return(&core.Dogu{Name: "test", Version: "1.0.0"}, nil, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
			},
			resource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to delete development dogu map from cluster",
			fields: fields{
				localDoguDescriptorRepoFn: func(t *testing.T) localDoguDescriptorRepository {
					mck := newMockLocalDoguDescriptorRepository(t)
					mck.EXPECT().Get(testCtx, dogu.SimpleNameVersion{Name: "test", Version: core.Version{Raw: "1.0.0", Major: 1}}).Return(nil, errors.NewNotFoundError(assert.AnError))
					mck.EXPECT().Add(testCtx, dogu.SimpleName("test"), &core.Dogu{Name: "test", Version: "1.0.0"}).Return(nil)
					return mck
				},
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
						Spec: v2.DoguSpec{
							Version: "1.0.0",
						},
					}).Return(&core.Dogu{Name: "test", Version: "1.0.0"}, &v2.DevelopmentDoguMap{}, nil)
					return mck
				},
				clientFn: func(t *testing.T) k8sClient {
					mck := newMockK8sClient(t)
					mck.EXPECT().Delete(testCtx, &v3.ConfigMap{}).Return(assert.AnError)
					return mck
				},
			},
			resource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FetchRemoteDoguDescriptorStep{
				client:                  tt.fields.clientFn(t),
				resourceDoguFetcher:     tt.fields.resourceDoguFetcherFn(t),
				localDoguDescriptorRepo: tt.fields.localDoguDescriptorRepoFn(t),
			}
			assert.Equalf(t, tt.want, f.Run(testCtx, tt.resource), "Run(%v, %v)", testCtx, tt.resource)
		})
	}
}
