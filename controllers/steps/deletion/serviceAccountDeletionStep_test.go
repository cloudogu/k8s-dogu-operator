package deletion

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewServiceAccountRemoverStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewServiceAccountRemoverStep(
			newMockK8sClient(t),
			&util.ManagerSet{
				LocalDoguFetcher:    cesregistry.NewLocalDoguFetcher(nil, nil),
				CommandExecutor:     exec.NewCommandExecutor(nil, nil, nil),
				ResourceDoguFetcher: cesregistry.NewResourceDoguFetcher(nil, nil),
				ClientSet:           nil,
			},
			util.ConfigRepositories{
				SensitiveDoguRepository: &repository.DoguConfigRepository{},
			},
			&config.OperatorConfig{
				Namespace: namespace,
			},
		)

		assert.NotNil(t, step)
	})
}

func TestServiceAccountRemoverStep_Run(t *testing.T) {
	tests := []struct {
		serviceAccountRemoverFn func(t *testing.T) serviceAccountRemover
		resourceDoguFetcherFn   func(t *testing.T) resourceDoguFetcher
		name                    string
		doguResource            *v2.Dogu
		want                    steps.StepResult
	}{
		{
			name: "should fail to fetch remote dogu descriptor",
			serviceAccountRemoverFn: func(t *testing.T) serviceAccountRemover {
				return newMockServiceAccountRemover(t)
			},
			resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
				resourceDoguFetcherMock := newMockResourceDoguFetcher(t)
				resourceDoguFetcherMock.EXPECT().FetchWithResource(testCtx, &v2.Dogu{}).Return(nil, nil, assert.AnError)
				return resourceDoguFetcherMock
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
			resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
				resourceDoguFetcherMock := newMockResourceDoguFetcher(t)
				resourceDoguFetcherMock.EXPECT().FetchWithResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{}, nil, nil)
				return resourceDoguFetcherMock
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
			resourceDoguFetcherFn: func(t *testing.T) resourceDoguFetcher {
				resourceDoguFetcherMock := newMockResourceDoguFetcher(t)
				resourceDoguFetcherMock.EXPECT().FetchWithResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{}, nil, nil)
				return resourceDoguFetcherMock
			},
			doguResource: &v2.Dogu{},
			want:         steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sas := &ServiceAccountRemoverStep{
				serviceAccountRemover: tt.serviceAccountRemoverFn(t),
				resourceDoguFetcher:   tt.resourceDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, sas.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
