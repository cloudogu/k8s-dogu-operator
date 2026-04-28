package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v3 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewNetworkPoliciesStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewNetworkPoliciesStep(newMockResourceUpserter(t), newMockLocalDoguFetcher(t), newMockImageRegistry(t), newMockServiceInterface(t))

		assert.NotEmpty(t, step)
	})
}

func TestNetworkPoliciesStep_Run(t *testing.T) {
	type fields struct {
		netPolUpserterFn   func(t *testing.T) netPolUpserter
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		imageRegistryFn    func(t *testing.T) imageRegistry
		serviceInterfaceFn func(t *testing.T) serviceInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to fetch dogu descriptor",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					return newMockNetPolUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					return newMockImageRegistry(t)
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					return newMockServiceInterface(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to fetch dogu service",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					return newMockNetPolUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					return newMockImageRegistry(t)
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get dogu service for \"test\": %w", assert.AnError)),
		},
		{
			name: "should fail to pull image config",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					return newMockNetPolUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(nil, assert.AnError)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to pull dogu image config for \"test\": %w", assert.AnError)),
		},
		{
			name: "should fail to collect dogu routes",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					return newMockNetPolUpserter(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(imageConfigFileWithInvalidRoutes(), nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to collect dogu routes for \"test\": %w", fmt.Errorf("failed to get service tags: failed to get service variables from environment variables: failed to split environment variable: environment variable [SERVICE_TAGS-invalidEnvironmentVariable] needs to be in form NAME=VALUE"))),
		},
		{
			name: "should fail to upsert network policy for dogu",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					mck := newMockNetPolUpserter(t)
					mck.EXPECT().UpsertDoguNetworkPolicies(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}, &cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, false).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(imageConfigFileWithoutRoutes(), nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to upsert network policy for dogu",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					mck := newMockNetPolUpserter(t)
					mck.EXPECT().UpsertDoguNetworkPolicies(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}, &cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, false).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test", Image: "test", Version: "1.0.0"}, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(imageConfigFileWithoutRoutes(), nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nps := &NetworkPoliciesStep{
				netPolUpserter:   tt.fields.netPolUpserterFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				imageRegistry:    tt.fields.imageRegistryFn(t),
				serviceInterface: tt.fields.serviceInterfaceFn(t),
			}
			got := nps.Run(testCtx, tt.doguResource)
			assert.Equalf(t, tt.want.RequeueAfter, got.RequeueAfter, "Run(%v, %v)", testCtx, tt.doguResource)
			assert.Equalf(t, tt.want.Continue, got.Continue, "Run(%v, %v)", testCtx, tt.doguResource)
			if tt.want.Err == nil {
				assert.NoError(t, got.Err)
			} else {
				assert.EqualError(t, got.Err, tt.want.Err.Error())
			}
		})
	}
}

func imageConfigFileWithoutRoutes() *v3.ConfigFile {
	return &v3.ConfigFile{Config: v3.Config{}}
}

func imageConfigFileWithInvalidRoutes() *v3.ConfigFile {
	return &v3.ConfigFile{
		Config: v3.Config{
			Env: []string{
				"SERVICE_TAGS-invalidEnvironmentVariable",
			},
		},
	}
}
