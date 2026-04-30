package deletion

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestNewDeleteExposedPortsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		localDoguFetcher := newMockLocalDoguFetcher(t)
		mapInterface := newMockConfigMapInterface(t)
		step := NewDeleteExposedPortsStep(localDoguFetcher, mapInterface)

		assert.Same(t, localDoguFetcher, step.localDoguFetcher)
	})
}

func TestDeleteExposedPortsStep_Run(t *testing.T) {
	type fields struct {
		localDoguFetcherFn    func(t *testing.T) localDoguFetcher
		exposedPortsManagerFn func(t *testing.T) exposedPortsManager
	}
	type args struct {
		doguResource *v2.Dogu
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   steps.StepResult
	}{
		{
			name: "should fail to get local dogu",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(nil, assert.AnError)
					return mck
				},
				exposedPortsManagerFn: func(t *testing.T) exposedPortsManager {
					return newMockExposedPortsManager(t)
				},
			},
			args: args{
				doguResource: &v2.Dogu{},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to delete ports",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{ExposedPorts: []core.ExposedPort{}}, nil)
					return mck
				},
				exposedPortsManagerFn: func(t *testing.T) exposedPortsManager {
					mck := newMockExposedPortsManager(t)
					mck.EXPECT().DeletePorts(testCtx, []core.ExposedPort{}).Return(nil, assert.AnError)
					return mck
				},
			},
			args: args{
				doguResource: &v2.Dogu{},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "successfully delete ports",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, &v2.Dogu{}).Return(&core.Dogu{ExposedPorts: []core.ExposedPort{}}, nil)
					return mck
				},
				exposedPortsManagerFn: func(t *testing.T) exposedPortsManager {
					mck := newMockExposedPortsManager(t)
					mck.EXPECT().DeletePorts(testCtx, []core.ExposedPort{}).Return(&v1.ConfigMap{}, nil)
					return mck
				},
			},
			args: args{
				doguResource: &v2.Dogu{},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eps := &DeleteExposedPortsStep{
				localDoguFetcher:    tt.fields.localDoguFetcherFn(t),
				exposedPortsManager: tt.fields.exposedPortsManagerFn(t),
			}
			assert.Equalf(t, tt.want, eps.Run(testCtx, tt.args.doguResource), "Run(%v, %v)", testCtx, tt.args.doguResource)
		})
	}
}
