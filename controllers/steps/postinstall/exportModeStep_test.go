package postinstall

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestNewExportModeStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		manager := newMockExportManager(t)

		step := NewExportModeStep(manager)

		assert.NotNil(t, step)
	})
}

func TestExportModeStep_Run(t *testing.T) {
	tests := []struct {
		name            string
		exportManagerFn func(t *testing.T) exportManager
		doguResource    *v2.Dogu
		want            steps.StepResult
	}{
		{
			name: "should fail to update export mode",
			exportManagerFn: func(t *testing.T) exportManager {
				mck := newMockExportManager(t)
				mck.EXPECT().UpdateExportMode(testCtx, &v2.Dogu{}).Return(assert.AnError)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.RequeueWithError(assert.AnError),
		},
		{
			name: "should succeed to update export mode",
			exportManagerFn: func(t *testing.T) exportManager {
				mck := newMockExportManager(t)
				mck.EXPECT().UpdateExportMode(testCtx, &v2.Dogu{}).Return(nil)
				return mck
			},
			doguResource: &v2.Dogu{},
			want:         steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ems := &ExportModeStep{
				exportManager: tt.exportManagerFn(t),
			}
			assert.Equalf(t, tt.want, ems.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
