package migration

import (
	"testing"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// migratableDogu returns a dogu whose v2 status allows the migration.
func migratableDogu() *v3beta1.Dogu {
	dogu := testDogu()
	dogu.Status = v3beta1.DoguStatus{
		Status: doguv2.DoguStatusInstalled,
		Conditions: []metav1.Condition{
			{Type: doguv2.ConditionSupportMode, Status: metav1.ConditionFalse},
			{Type: doguv2.ConditionMeetsMinVolumeSize, Status: metav1.ConditionTrue},
		},
	}
	return dogu
}

func setCondition(dogu *v3beta1.Dogu, conditionType string, status metav1.ConditionStatus) {
	for i := range dogu.Status.Conditions {
		if dogu.Status.Conditions[i].Type == conditionType {
			dogu.Status.Conditions[i].Status = status
		}
	}
}

func TestMigrationPreconditionsStep_Run(t *testing.T) {
	t.Run("continues if the v2 dogu can be migrated", func(t *testing.T) {
		dogu := migratableDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)

		result := NewMigrationPreconditionsStep(detector, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("continues without checks if the migration is already in progress", func(t *testing.T) {
		dogu := migratableDogu()
		dogu.Status.ExportMode = true
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.InProgress, nil)

		result := NewMigrationPreconditionsStep(detector, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("continues without checks if no migration is required", func(t *testing.T) {
		dogu := migratableDogu()
		dogu.Status.ExportMode = true
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.NotRequired, nil)

		result := NewMigrationPreconditionsStep(detector, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	blockedTests := []struct {
		name    string
		modify  func(dogu *v3beta1.Dogu)
		wantMsg string
	}{
		{
			name:    "active support mode",
			modify:  func(dogu *v3beta1.Dogu) { setCondition(dogu, doguv2.ConditionSupportMode, metav1.ConditionTrue) },
			wantMsg: "The migration to v3 is blocked by the v2 dogu: support mode is active",
		},
		{
			name:    "active export mode",
			modify:  func(dogu *v3beta1.Dogu) { dogu.Status.ExportMode = true },
			wantMsg: "The migration to v3 is blocked by the v2 dogu: export mode is active",
		},
		{
			name:    "running v2 operation",
			modify:  func(dogu *v3beta1.Dogu) { dogu.Status.Status = doguv2.DoguStatusUpgrading },
			wantMsg: `The migration to v3 is blocked by the v2 dogu: operation "upgrading" is still running`,
		},
		{
			name: "running volume resize",
			modify: func(dogu *v3beta1.Dogu) {
				setCondition(dogu, doguv2.ConditionMeetsMinVolumeSize, metav1.ConditionFalse)
			},
			wantMsg: "The migration to v3 is blocked by the v2 dogu: data volume is being resized",
		},
		{
			name: "several unmet preconditions",
			modify: func(dogu *v3beta1.Dogu) {
				setCondition(dogu, doguv2.ConditionSupportMode, metav1.ConditionTrue)
				dogu.Status.ExportMode = true
				dogu.Status.Status = doguv2.DoguStatusChangingExportMode
			},
			wantMsg: `The migration to v3 is blocked by the v2 dogu: support mode is active, export mode is active, operation "changing export-mode" is still running`,
		},
	}
	for _, tt := range blockedTests {
		t.Run("keeps the dogu pending on "+tt.name, func(t *testing.T) {
			dogu := migratableDogu()
			tt.modify(dogu)
			detector := NewMockMigrationDetector(t)
			detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
			recorder := NewMockEventRecorder(t)
			recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, tt.wantMsg)

			result := NewMigrationPreconditionsStep(detector, recorder).Run(testCtx, dogu)

			assert.Equal(t, stepsv3.RequeueAfter(time.Minute, v3beta1.ReasonMigrationPending, tt.wantMsg), result)
		})
	}

	t.Run("detection error requeues without event", func(t *testing.T) {
		dogu := migratableDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.NotRequired, assert.AnError)

		result := NewMigrationPreconditionsStep(detector, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, `failed to detect migration state of dogu "nexus"`)
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
		assert.False(t, result.Continue)
	})
}
