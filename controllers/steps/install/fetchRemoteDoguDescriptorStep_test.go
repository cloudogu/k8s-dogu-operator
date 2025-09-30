package install

import (
	"context"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestNewFetchRemoteDoguDescriptorStep(t *testing.T) {
	step := NewFetchRemoteDoguDescriptorStep(newMockK8sClient(t), newMockLocalDoguDescriptorRepository(t), newMockResourceDoguFetcher(t))
	assert.NotEmpty(t, step)
}

func TestFetchRemoteDoguDescriptorStep_Run(t *testing.T) {
	t.Fatal("not implemented")
	type fields struct {
		client                  k8sClient
		resourceDoguFetcher     resourceDoguFetcher
		localDoguDescriptorRepo localDoguDescriptorRepository
	}
	type args struct {
		ctx      context.Context
		resource *v2.Dogu
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   steps.StepResult
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FetchRemoteDoguDescriptorStep{
				client:                  tt.fields.client,
				resourceDoguFetcher:     tt.fields.resourceDoguFetcher,
				localDoguDescriptorRepo: tt.fields.localDoguDescriptorRepo,
			}
			assert.Equalf(t, tt.want, f.Run(tt.args.ctx, tt.args.resource), "Run(%v, %v)", tt.args.ctx, tt.args.resource)
		})
	}
}
