package install

import (
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewServiceAccountStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewServiceAccountStep(
			&util.ManagerSet{
				LocalDoguFetcher:      newMockLocalDoguFetcher(t),
				ServiceAccountCreator: newMockServiceAccountCreator(t),
			},
		)

		assert.NotNil(t, step)
	})
}

func TestServiceAccountStep_Run(t *testing.T) {
	type fields struct {
		serviceAccountCreatorFn func(t *testing.T) serviceAccountCreator
		localDoguFetcherFn      func(t *testing.T) localDoguFetcher
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get dogu descriptor",
			fields: fields{
				serviceAccountCreatorFn: func(t *testing.T) serviceAccountCreator {
					return newMockServiceAccountCreator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
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
			name: "should fail to create service accounts for dogu",
			fields: fields{
				serviceAccountCreatorFn: func(t *testing.T) serviceAccountCreator {
					mck := newMockServiceAccountCreator(t)
					mck.EXPECT().CreateAll(testCtx, &core.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
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
			name: "should succeed to create service accounts for dogu",
			fields: fields{
				serviceAccountCreatorFn: func(t *testing.T) serviceAccountCreator {
					mck := newMockServiceAccountCreator(t)
					mck.EXPECT().CreateAll(testCtx, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
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
			sas := &ServiceAccountStep{
				serviceAccountCreator: tt.fields.serviceAccountCreatorFn(t),
				localDoguFetcher:      tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, sas.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
