package upgrade

import (
	"fmt"
	"testing"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestNewRegisterDoguVersionStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewRegisterDoguVersionStep(newMockLocalDoguFetcher(t), newMockDoguRegistrator(t))

		assert.NotNil(t, step)
	})
}

func TestRegisterDoguVersionStep_Run(t *testing.T) {
	type fields struct {
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		doguRegistratorFn  func(t *testing.T) doguRegistrator
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
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(nil, assert.AnError)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					return newMockDoguRegistrator(t)
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should fail to register dogu",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{}, nil)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					mck := newMockDoguRegistrator(t)
					mck.EXPECT().RegisterDoguVersion(testCtx, &core.Dogu{}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", assert.AnError)),
		},
		{
			name: "should continue if dogu version is already registered",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{}, nil)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					mck := newMockDoguRegistrator(t)
					mck.EXPECT().RegisterDoguVersion(testCtx, &core.Dogu{}).Return(cloudoguerrors.NewAlreadyExistsError(assert.AnError))
					return mck
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.Continue(),
		},
		{
			name: "should continue if dogu version is successfully registered",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{}, nil)
					return mck
				},
				doguRegistratorFn: func(t *testing.T) doguRegistrator {
					mck := newMockDoguRegistrator(t)
					mck.EXPECT().RegisterDoguVersion(testCtx, &core.Dogu{}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdvs := &RegisterDoguVersionStep{
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				doguRegistrator:  tt.fields.doguRegistratorFn(t),
			}
			assert.Equalf(t, tt.want, rdvs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
