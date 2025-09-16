package install

import (
	"fmt"
	"testing"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewVolumeGeneratorStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewVolumeGeneratorStep(newMockLocalDoguFetcher(t), newMockResourceUpserter(t), newMockPersistentVolumeClaimInterface(t))

		assert.NotNil(t, step)
	})
}

func TestVolumeGeneratorStep_Run(t *testing.T) {
	type fields struct {
		localDoguFetcherFn func(t *testing.T) localDoguFetcher
		resourceUpserterFn func(t *testing.T) resourceUpserter
		pvcGetterFn        func(t *testing.T) persistentVolumeClaimInterface
	}
	tests := []struct {
		name         string
		fields       fields
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should fail to get pvcs",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceUpserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				pvcGetterFn: func(t *testing.T) persistentVolumeClaimInterface {
					mck := newMockPersistentVolumeClaimInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, assert.AnError)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "pvs already exist",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					return newMockLocalDoguFetcher(t)
				},
				resourceUpserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				pvcGetterFn: func(t *testing.T) persistentVolumeClaimInterface {
					mck := newMockPersistentVolumeClaimInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, nil)
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.Continue(),
		},
		{
			name: "should not find any pvcs for dogu",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(nil, assert.AnError)
					return mck
				},
				resourceUpserterFn: func(t *testing.T) resourceUpserter {
					return newMockResourceUpserter(t)
				},
				pvcGetterFn: func(t *testing.T) persistentVolumeClaimInterface {
					mck := newMockPersistentVolumeClaimInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", "test", assert.AnError)),
		},
		{
			name: "should fail to upsert pvc",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceUpserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguPVCs(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{}).Return(nil, assert.AnError)
					return mck
				},
				pvcGetterFn: func(t *testing.T) persistentVolumeClaimInterface {
					mck := newMockPersistentVolumeClaimInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					return mck
				},
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should successfully upsert pvc",
			fields: fields{
				localDoguFetcherFn: func(t *testing.T) localDoguFetcher {
					mck := newMockLocalDoguFetcher(t)
					mck.EXPECT().FetchInstalled(testCtx, dogu.SimpleName("test")).Return(&core.Dogu{}, nil)
					return mck
				},
				resourceUpserterFn: func(t *testing.T) resourceUpserter {
					mck := newMockResourceUpserter(t)
					mck.EXPECT().UpsertDoguPVCs(testCtx, &v2.Dogu{
						ObjectMeta: v1.ObjectMeta{Name: "test"},
					}, &core.Dogu{}).Return(nil, nil)
					return mck
				},
				pvcGetterFn: func(t *testing.T) persistentVolumeClaimInterface {
					mck := newMockPersistentVolumeClaimInterface(t)
					mck.EXPECT().Get(testCtx, "test", v1.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
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
			vgs := &VolumeGeneratorStep{
				localDoguFetcher: tt.fields.localDoguFetcherFn(t),
				resourceUpserter: tt.fields.resourceUpserterFn(t),
				pvcGetter:        tt.fields.pvcGetterFn(t),
			}
			assert.Equalf(t, tt.want, vgs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
