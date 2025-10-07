package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCustomK8sResourceStep_Run(t *testing.T) {
	type fields struct {
		recorderFn         func(t *testing.T) eventRecorder
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		execPodFactoryFn   func(t *testing.T) execPodFactory
		fileExtractorFn    func(t *testing.T) fileExtractor
		collectApplierFn   func(t *testing.T) collectApplier
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
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					return newMockExecPodFactory(t)
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					return newMockFileExtractor(t)
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					return newMockCollectApplier(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", assert.AnError)),
		},
		{
			name: "should continue if exec pod does not exist",
			fields: fields{
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(false)
					return mck
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					return newMockFileExtractor(t)
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					return newMockCollectApplier(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
		{
			name: "should fail on check ready",
			fields: fields{
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(assert.AnError)
					return mck
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					return newMockFileExtractor(t)
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					return newMockCollectApplier(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to check if exec pod is ready: %w", assert.AnError)),
		},
		{
			name: "should fail to extract custom resources",
			fields: fields{
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					mck := newMockFileExtractor(t)
					mck.EXPECT().ExtractK8sResourcesFromExecPod(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(nil, assert.AnError)
					return mck
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					return newMockCollectApplier(t)
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to extract customK8sResources: %w", assert.AnError)),
		},
		{
			name: "should fail to apply custom resources",
			fields: fields{
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					mck := newMockFileExtractor(t)
					mck.EXPECT().ExtractK8sResourcesFromExecPod(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(map[string]string{}, nil)
					return mck
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					mck := newMockCollectApplier(t)
					mck.EXPECT().CollectApply(testCtx, map[string]string{}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to apply customK8sResources: %w", assert.AnError)),
		},
		{
			name: "should successfully apply custom resources",
			fields: fields{
				recorderFn: func(t *testing.T) eventRecorder {
					return newMockEventRecorder(t)
				},
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{Name: "test"}, nil)
					return mck
				},
				execPodFactoryFn: func(t *testing.T) execPodFactory {
					mck := newMockExecPodFactory(t)
					mck.EXPECT().Exists(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(true)
					mck.EXPECT().CheckReady(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(nil)
					return mck
				},
				fileExtractorFn: func(t *testing.T) fileExtractor {
					mck := newMockFileExtractor(t)
					mck.EXPECT().ExtractK8sResourcesFromExecPod(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{Name: "test"}).Return(map[string]string{}, nil)
					return mck
				},
				collectApplierFn: func(t *testing.T) collectApplier {
					mck := newMockCollectApplier(t)
					mck.EXPECT().CollectApply(testCtx, map[string]string{}, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}).Return(nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ses := &CustomK8sResourceStep{
				recorder:         tt.fields.recorderFn(t),
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				execPodFactory:   tt.fields.execPodFactoryFn(t),
				fileExtractor:    tt.fields.fileExtractorFn(t),
				collectApplier:   tt.fields.collectApplierFn(t),
			}
			assert.Equalf(t, tt.want, ses.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func TestNewCustomK8sResourceStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		recorder := newMockEventRecorder(t)
		fetcher := newMockLocalDoguFetcher(t)
		factory := newMockExecPodFactory(t)
		extractor := newMockFileExtractor(t)
		applier := newMockCollectApplier(t)

		step := NewCustomK8sResourceStep(recorder, fetcher, factory, extractor, applier)

		assert.Same(t, recorder, step.recorder)
		assert.Same(t, fetcher, step.localDoguFetcher)
		assert.Same(t, factory, step.execPodFactory)
		assert.Same(t, extractor, step.fileExtractor)
		assert.Same(t, applier, step.collectApplier)
	})
}
