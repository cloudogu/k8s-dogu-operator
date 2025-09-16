package postinstall

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestAdditionalMountsStep_Run(t *testing.T) {
	tests := []struct {
		name                     string
		additionalMountManagerFn func(t *testing.T) additionalMountManager
		doguResource             *v2.Dogu
		want                     steps.StepResult
	}{
		{
			name: "should fail on check if additional mounts changed",
			additionalMountManagerFn: func(t *testing.T) additionalMountManager {
				mck := newMockAdditionalMountManager(t)
				mck.EXPECT().AdditionalMountsChanged(testCtx, &v2.Dogu{}).Return(false, assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should fail on update of additional mounts",
			additionalMountManagerFn: func(t *testing.T) additionalMountManager {
				mck := newMockAdditionalMountManager(t)
				mck.EXPECT().AdditionalMountsChanged(testCtx, &v2.Dogu{}).Return(true, nil)
				mck.EXPECT().UpdateAdditionalMounts(testCtx, &v2.Dogu{}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should successfully update additional mounts",
			additionalMountManagerFn: func(t *testing.T) additionalMountManager {
				mck := newMockAdditionalMountManager(t)
				mck.EXPECT().AdditionalMountsChanged(testCtx, &v2.Dogu{}).Return(true, nil)
				mck.EXPECT().UpdateAdditionalMounts(testCtx, &v2.Dogu{}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ams := &AdditionalMountsStep{
				additionalMountManager: tt.additionalMountManagerFn(t),
			}
			assert.Equalf(t, tt.want, ams.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func TestNewAdditionalMountsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		manager := newMockAdditionalMountManager(t)

		step := NewAdditionalMountsStep(manager)

		assert.NotNil(t, step)
	})
}
