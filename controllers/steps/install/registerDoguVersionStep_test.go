package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewRegisterDoguVersionStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRegisterDoguVersionStep(
			&util.ManagerSet{
				ResourceDoguFetcher: newMockResourceDoguFetcher(t),
				LocalDoguFetcher:    newMockLocalDoguFetcher(t),
				DoguRegistrator:     newMockDoguRegistrator(t),
			},
		)

		assert.NotNil(t, step)
	})
}

func TestRegisterDoguVersionStep_Run(t *testing.T) {
	type fields struct {
		resourceDoguFetcherFn func(t *testing.T) resourceDoguFetcher
		doguRegistratorFn     func(t *testing.T) doguRegistrator
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to check if dogu is enabled",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					return newMockDoguRegistrator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().Enabled(testCtx, dogu.SimpleName("test")).Return(false, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to check if dogu is enabled: %w", assert.AnError)),
		},
		{
			name: "should continue if dogu is enabled",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					return newMockResourceDoguFetcher(t)
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					return newMockDoguRegistrator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().Enabled(testCtx, dogu.SimpleName("test")).Return(true, nil)
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
		{
			name: "should fail to fetch dogu descriptor if dogu is not enabled",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}).Return(nil, nil, assert.AnError)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					return newMockDoguRegistrator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().Enabled(testCtx, dogu.SimpleName("test")).Return(false, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should fail to register dogu if dogu is not enabled",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}).Return(&cesappcore.Dogu{Name: "test"}, nil, nil)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					mck := newMockDoguRegistrator(t)
					mck.EXPECT().RegisterNewDogu(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}, &cesappcore.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().Enabled(testCtx, dogu.SimpleName("test")).Return(false, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", assert.AnError)),
		},
		{
			name: "should succeed to register dogu if dogu is not enabled",
			fields: fields{
				resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
					mck := newMockResourceDoguFetcher(t)
					mck.EXPECT().FetchWithResource(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}).Return(&cesappcore.Dogu{Name: "test"}, nil, nil)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					mck := newMockDoguRegistrator(t)
					mck.EXPECT().RegisterNewDogu(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{
							Name: "test",
						},
					}, &cesappcore.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().Enabled(testCtx, dogu.SimpleName("test")).Return(false, nil)
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
			rdvs := &RegisterDoguVersionStep{
				resourceDoguFetcher: tt.fields.resourceDoguFetcherFn(t),
				doguRegistrator:     tt.fields.doguRegistratorFn(t),
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, rdvs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
