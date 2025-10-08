package deletion

import (
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceAccountRemoverStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		// given
		remover := newMockServiceAccountRemover(t)
		fetcher := newMockLocalDoguFetcher(t)

		// when
		step := NewServiceAccountRemoverStep(remover, fetcher)

		// then
		assert.NotEmpty(t, step)
	})
}

func TestServiceAccountRemoverStep_Run(t *testing.T) {
	tests := []struct {
		serviceAccountRemoverFn func(t *testing.T) serviceAccountRemover
		localDoguFetcherFn      func(t *testing.T) localDoguFetcher
		name                    string
		doguResource            *v2.Dogu
		want                    steps.StepResult
	}{
		{
			name: "should fail to fetch remote dogu descriptor",
			serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
				return newMockServiceAccountRemover(t)
			},
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				localDoguFetcherMock := newMockLocalDoguFetcher(t)
				localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(nil, assert.AnError)
				return localDoguFetcherMock
			},
			doguResource: &v2.Dogu{},
			want:         steps.StepResult{Err: assert.AnError},
		},
		{
			name: "should fail to remove service accounts of dogu",
			serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
				serviceAccountRemoverMock := newMockServiceAccountRemover(t)
				serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, &core.Dogu{}).Return(assert.AnError)
				return serviceAccountRemoverMock
			},
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				localDoguFetcherMock := newMockLocalDoguFetcher(t)
				localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&core.Dogu{}, nil)
				return localDoguFetcherMock
			},
			doguResource: &v2.Dogu{},
			want:         steps.StepResult{Err: assert.AnError},
		},
		{
			name: "should remove service accounts of dogu",
			serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
				serviceAccountRemoverMock := newMockServiceAccountRemover(t)
				serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, &core.Dogu{}).Return(nil)
				return serviceAccountRemoverMock
			},
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				localDoguFetcherMock := newMockLocalDoguFetcher(t)
				localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(&core.Dogu{}, nil)
				return localDoguFetcherMock
			},
			doguResource: &v2.Dogu{},
			want:         steps.StepResult{Continue: true},
		},
		{
			name: "should continue if dogu is not found",
			serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
				return newMockServiceAccountRemover(t)
			},
			localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
				localDoguFetcherMock := newMockLocalDoguFetcher(t)
				localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("")).Return(nil, cloudoguerrors.NewNotFoundError(assert.AnError))
				return localDoguFetcherMock
			},
			doguResource: &v2.Dogu{},
			want:         steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sas := &ServiceAccountRemoverStep{
				serviceAccountRemover: tt.serviceAccountRemoverFn(t),
				doguFetcher:           tt.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, sas.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
