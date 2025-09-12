package postinstall

import (
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewVolumeExpanderStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		doguInterfaceMock := newMockDoguInterface(t)
		ecosystemInterfaceMock := newMockEcosystemInterface(t)
		ecosystemInterfaceMock.EXPECT().Dogus(namespace).Return(doguInterfaceMock)

		step := NewVolumeExpanderStep(
			newMockK8sClient(t),
			&util.ManagerSet{
				LocalDoguFetcher: newMockLocalDoguFetcher(t),
				EcosystemClient:  ecosystemInterfaceMock,
			},
			namespace,
		)

		assert.NotNil(t, step)
	})
}

func TestVolumeExpanderStep_Run(t *testing.T) {
	type fields struct {
		clientFn           func(t *testing.T) k8sClient
		doguInterfaceFn    func(t *testing.T) doguInterface
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
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
				clientFn: func(t *testing.T) k8sClient {
					return newMockK8sClient(t)
				},
				doguInterfaceFn: func(t *testing.T) doguInterface {
					return newMockDoguInterface(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{ObjectMeta: v1.ObjectMeta{Name: "test"}},
			want:         steps.RequeueWithError(assert.AnError),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &VolumeExpanderStep{
				client:           tt.fields.clientFn(t),
				doguInterface:    tt.fields.doguInterfaceFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
			}
			assert.Equalf(t, tt.want, vs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
