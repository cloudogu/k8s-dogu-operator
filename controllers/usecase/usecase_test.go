package usecase

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/postinstall"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/upgrade"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func TestDoguUseCase_HandleUntilApplied(t *testing.T) {
	tests := []struct {
		name             string
		stepsFn          func(t *testing.T) []Step
		doguResource     *v2.Dogu
		wantRequeueAfter time.Duration
		wantContinue     bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "should requeue run on requeueAfter time",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueAfter(2))
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 2,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
		{
			name: "should requeue run on error",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.RequeueWithError(assert.AnError))
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.Error,
		},
		{
			name: "should continue after step",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Continue())
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     true,
			wantErr:          assert.NoError,
		},
		{
			name: "should abort after step",
			stepsFn: func(t *testing.T) []Step {
				step := NewMockStep(t)
				step.EXPECT().Run(testCtx, mock.Anything).Return(steps.Abort())
				return []Step{step}
			},
			doguResource: &v2.Dogu{
				ObjectMeta: v1.ObjectMeta{Name: "test"},
			},
			wantRequeueAfter: 0,
			wantContinue:     false,
			wantErr:          assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duc := &DoguUseCase{
				steps: tt.stepsFn(t),
			}
			got, got1, err := duc.HandleUntilApplied(testCtx, tt.doguResource)
			if !tt.wantErr(t, err, fmt.Sprintf("HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)) {
				return
			}
			assert.Equalf(t, tt.wantRequeueAfter, got, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
			assert.Equalf(t, tt.wantContinue, got1, "HandleUntilApplied(%v, %v)", testCtx, tt.doguResource)
		})
	}
}

func TestNewDoguDeleteUseCase(t *testing.T) {
	t.Run("should successfully create dogu delete use case with steps in correct order", func(t *testing.T) {
		statusStep := &deletion.StatusStep{}
		serviceAccountRemoverStep := &deletion.ServiceAccountRemoverStep{}
		deleteOutOfHealthConfigMapStep := &deletion.DeleteOutOfHealthConfigMapStep{}
		removeSensitiveDoguConfigStep := deletion.NewRemoveDoguConfigStep(nil)
		removeFinalizerStep := &deletion.RemoveFinalizerStep{}

		got := NewDoguDeleteUseCase(
			statusStep,
			serviceAccountRemoverStep,
			deleteOutOfHealthConfigMapStep,
			removeSensitiveDoguConfigStep,
			removeFinalizerStep,
		)

		wantTypes := []string{
			"*deletion.StatusStep",
			"*deletion.ServiceAccountRemoverStep",
			"*deletion.DeleteOutOfHealthConfigMapStep",
			"*deletion.removeDoguConfigStep",
			"*deletion.RemoveFinalizerStep",
		}

		assert.NotNil(t, got)
		require.True(t,
			slices.Equal(typesOf(got.steps), wantTypes),
			"order mismatch: got=%v want=%v",
			typesOf(got.steps), wantTypes,
		)
	})
}

func TestNewDoguInstallOrChangeUseCase(t *testing.T) {
	t.Run("should successfully create dogu install or change use case with steps in correct order", func(t *testing.T) {
		got := NewDoguInstallOrChangeUseCase(
			&install.InitializeConditionsStep{},
			&install.HealthCheckStep{},
			&install.FetchRemoteDoguDescriptorStep{},
			&install.ValidationStep{},
			&install.PauseReconciliationStep{},
			&install.CreateFinalizerStep{},
			install.NewCreateConfigStep(nil),
			install.NewOwnerReferenceStep(nil),
			install.NewCreateConfigStep(nil),
			install.NewOwnerReferenceStep(nil),
			&install.RegisterDoguVersionStep{},
			install.NewOwnerReferenceStep(nil),
			&install.ServiceAccountStep{},
			&install.ServiceStep{},
			&install.CreateExecPodStep{},
			&install.CustomK8sResourceStep{},
			&install.CreateVolumeStep{},
			&install.NetworkPoliciesStep{},
			&install.CreateDeploymentStep{},

			&postinstall.StartStopStep{},
			&postinstall.VolumeExpanderStep{},
			&postinstall.AdditionalIngressAnnotationsStep{},
			&postinstall.SecurityContextStep{},
			&postinstall.ExportModeStep{},
			&postinstall.SupportModeStep{},
			&postinstall.AdditionalMountsStep{},

			&upgrade.PreUpgradeStatusStep{},
			&upgrade.UpdateDeploymentVersionStep{},
			&upgrade.DeleteExecPodStep{},
			&upgrade.PostUpgradeStep{},
			&upgrade.RegenerateDeploymentStep{},
			&upgrade.RegisterDoguVersionStep{},
			&upgrade.InstalledVersionStep{},
			&upgrade.UpdateStartedAtStep{},
			&upgrade.RestartAfterConfigChangeStep{},
		)

		wantTypes := []string{
			"*install.InitializeConditionsStep",
			"*install.HealthCheckStep",
			"*install.FetchRemoteDoguDescriptorStep",
			"*install.ValidationStep",
			"*install.PauseReconciliationStep",
			"*install.CreateFinalizerStep",
			"*install.CreateConfigStep",
			"*install.OwnerReferenceStep",
			"*install.CreateConfigStep",
			"*install.OwnerReferenceStep",
			"*install.RegisterDoguVersionStep",
			"*install.OwnerReferenceStep",
			"*install.ServiceAccountStep",
			"*install.ServiceStep",
			"*install.CreateExecPodStep",
			"*install.CustomK8sResourceStep",
			"*install.CreateVolumeStep",
			"*install.NetworkPoliciesStep",
			"*install.CreateDeploymentStep",

			"*postinstall.StartStopStep",
			"*postinstall.VolumeExpanderStep",
			"*postinstall.AdditionalIngressAnnotationsStep",
			"*postinstall.SecurityContextStep",
			"*postinstall.ExportModeStep",
			"*postinstall.SupportModeStep",
			"*postinstall.AdditionalMountsStep",

			"*upgrade.PreUpgradeStatusStep",
			"*upgrade.UpdateDeploymentVersionStep",
			"*upgrade.DeleteExecPodStep",
			"*upgrade.PostUpgradeStep",
			"*upgrade.InstalledVersionStep",
			"*upgrade.RegenerateDeploymentStep",
			"*upgrade.RegisterDoguVersionStep",
			"*upgrade.UpdateStartedAtStep",
			"*upgrade.RestartAfterConfigChangeStep",
		}

		assert.NotNil(t, got)
		require.True(t,
			slices.Equal(typesOf(got.steps), wantTypes),
			"order mismatch: got=%v want=%v",
			typesOf(got.steps), wantTypes,
		)
	})
}

func typesOf[T any](xs []T) []string {
	out := make([]string, len(xs))
	for i, v := range xs {
		out[i] = reflect.TypeOf(v).String()
	}
	return out
}
