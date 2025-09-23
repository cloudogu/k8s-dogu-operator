package install

import (
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
)

func TestNewPauseReconcilationStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewPauseReconcilationStep()

		assert.NotNil(t, step)
	})
}

func TestPauseReconcilationStep_Run(t *testing.T) {
	tests := []struct {
		name         string
		doguResource *v2.Dogu
		want         steps.StepResult
	}{
		{
			name: "should abort because of active pause reconcilation",
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					PauseReconcilation: true,
				},
			},
			want: steps.Abort(),
		},
		{
			name: "should continue because of inactive pause reconcilation",
			doguResource: &v2.Dogu{
				Spec: v2.DoguSpec{
					PauseReconcilation: false,
				},
			},
			want: steps.Continue(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs := &PauseReconcilationStep{}
			assert.Equalf(t, tt.want, prs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
