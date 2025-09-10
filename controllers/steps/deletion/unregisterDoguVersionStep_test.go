package deletion

import (
	"fmt"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewUnregisterDoguVersionStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewUnregisterDoguVersionStep(newMockDoguRegistrator(t))

		assert.NotNil(t, step)
	})
}

func TestUnregisterDoguVersionStep_Run(t *testing.T) {
	tests := []struct {
		name              string
		doguRegistratorFn func(t *testing.T) doguRegistrator
		doguResource      *v2.Dogu
		want              steps.StepResult
	}{
		{
			name: "should fail to unregister dogu",
			doguRegistratorFn: func(t *testing.T) doguRegistrator {
				registratorMock := newMockDoguRegistrator(t)
				registratorMock.EXPECT().UnregisterDogu(testCtx, "test").Return(assert.AnError)
				return registratorMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.StepResult{Err: fmt.Errorf("failed to register dogu: %w", assert.AnError)},
		},
		{
			name: "should unregister dogu",
			doguRegistratorFn: func(t *testing.T) doguRegistrator {
				registratorMock := newMockDoguRegistrator(t)
				registratorMock.EXPECT().UnregisterDogu(testCtx, "test").Return(nil)
				return registratorMock
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			want: steps.StepResult{Continue: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			udvs := &UnregisterDoguVersionStep{
				doguRegistrator: tt.doguRegistratorFn(t),
			}
			assert.Equalf(t, tt.want, udvs.Run(testCtx, tt.doguResource), "Run(%v, %v)", testCtx, tt.doguResource)
		})
	}
}
