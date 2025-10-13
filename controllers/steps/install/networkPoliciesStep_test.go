package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewNetworkPoliciesStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewNetworkPoliciesStep(newMockResourceUpserter(t), newMockLocalDoguFetcher(t), newMockServiceInterface(t))

		assert.NotEmpty(t, step)
	})
}

func TestNetworkPoliciesStep_Run(t *testing.T) {
	type fields struct {
		netPolUpserterFn   func(t *testing.T) netPolUpserter
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
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
			name: "should fail to upsert network policy for dogu",
			fields: fields{
				netPolUpserterFn: func(t *testing.T) netPolUpserter {
					mck := newMockNetPolUpserter(t)
					mck.EXPECT().UpsertDoguNetworkPolicies(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}, &cesappcore.Dogu{Name: "test"}, &coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test"}, nil)
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
					}, &cesappcore.Dogu{Name: "test"}, &coreV1.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&cesappcore.Dogu{Name: "test"}, nil)
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
				serviceInterface: tt.fields.serviceInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, nps.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
