package install

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v3 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	v4 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewServiceStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewServiceStep(newMockImageRegistry(t), newMockServiceInterface(t), newMockServiceGenerator(t), newMockLocalDoguFetcher(t))

		assert.NotNil(t, step)
	})
}

func TestServiceStep_Run(t *testing.T) {

	testDoguCR := &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{Name: "test"},
		Spec: v2.DoguSpec{
			Version: "1.0.0",
		},
	}
	testDogu := &core.Dogu{Name: "test", Image: "test", Version: "1.0.0"}

	type fields struct {
		serviceGeneratorFn func(t *testing.T) serviceGenerator
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
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					return newMockServiceGenerator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(nil, assert.AnError)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					return newMockImageRegistry(t)
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					return newMockServiceInterface(t)
				},
			},
			doguResource: testDoguCR,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to pull image config",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					return newMockServiceGenerator(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(nil, assert.AnError)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					return newMockServiceInterface(t)
				},
			},
			doguResource: testDoguCR,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to generate services",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					mck := newMockServiceGenerator(t)
					mck.EXPECT().CreateDoguService(
						testDoguCR,
						testDogu,
						&v3.ConfigFile{},
					).Return(nil, assert.AnError)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(&v3.ConfigFile{}, nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					return newMockServiceInterface(t)
				},
			},
			doguResource: testDoguCR,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to get service",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					mck := newMockServiceGenerator(t)
					mck.EXPECT().CreateDoguService(
						testDoguCR,
						testDogu,
						&v3.ConfigFile{},
					).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(&v3.ConfigFile{}, nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
				Spec: v2.DoguSpec{
					Version: "1.0.0",
				},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to upsert service if no service for dogu is found",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					mck := newMockServiceGenerator(t)
					mck.EXPECT().CreateDoguService(
						testDoguCR,
						testDogu,
						&v3.ConfigFile{},
					).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(&v3.ConfigFile{}, nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					mck.EXPECT().Create(testCtx, &v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, v1.CreateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: testDoguCR,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail to update service if service for dogu is found",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					mck := newMockServiceGenerator(t)
					mck.EXPECT().CreateDoguService(
						testDoguCR,
						testDogu,
						&v3.ConfigFile{},
					).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(&v3.ConfigFile{}, nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					mck.EXPECT().Update(testCtx, &v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, v1.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: testDoguCR,
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update service if service for dogu is found",
			fields: fields{
				serviceGeneratorFn: func(t *testing.T) serviceGenerator {
					mck := newMockServiceGenerator(t)
					mck.EXPECT().CreateDoguService(
						testDoguCR,
						testDogu,
						&v3.ConfigFile{},
					).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					return mck
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchForResource(testCtx, testDoguCR).Return(testDogu, nil)
					return mck
				},
				imageRegistryFn: func(t *testing.T) imageRegistry {
					mck := newMockImageRegistry(t)
					mck.EXPECT().PullImageConfig(testCtx, "test:1.0.0").Return(&v3.ConfigFile{}, nil)
					return mck
				},
				serviceInterfaceFn: func(t *testing.T) serviceInterface {
					mck := newMockServiceInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(&v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, nil)
					mck.EXPECT().Update(testCtx, &v4.Service{ObjectMeta: v1.ObjectMeta{Name: "test"}}, v1.UpdateOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: testDoguCR,
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ses := &ServiceStep{
				serviceGenerator: tt.fields.serviceGeneratorFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				imageRegistry:    tt.fields.imageRegistryFn(t),
				serviceInterface: tt.fields.serviceInterfaceFn(t),
			}
			assert.Equalf(t, tt.want, ses.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
