package migration

import (
	"context"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.Background()

func testDogu() *v3beta1.Dogu {
	return &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "nexus", Namespace: "ecosystem"},
		Spec:       v3beta1.DoguSpec{Name: "nexus", DoguNamespace: "official", DoguApiVersion: v3beta1.DoguApiVersionV3},
	}
}

func TestMigrationPendingStep_Run(t *testing.T) {
	t.Run("required migration keeps the dogu pending", func(t *testing.T) {
		dogu := testDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.Required, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, "V2 resources found, the dogu has to be migrated to v3")

		result := NewMigrationPendingStep(detector, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.RequeueAfter(time.Minute, v3beta1.ReasonMigrationPending, "V2 resources found, the dogu has to be migrated to v3"), result)
	})

	t.Run("migration in progress keeps the dogu pending", func(t *testing.T) {
		dogu := testDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.InProgress, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, "Migration resources found, the migration to v3 has to be resumed")

		result := NewMigrationPendingStep(detector, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.RequeueAfter(time.Minute, v3beta1.ReasonMigrationPending, "Migration resources found, the migration to v3 has to be resumed"), result)
	})

	t.Run("detection error requeues without event", func(t *testing.T) {
		dogu := testDogu()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, dogu).Return(dogumigration.NotRequired, assert.AnError)

		result := NewMigrationPendingStep(detector, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.ErrorIs(t, result.Err, assert.AnError)
		assert.ErrorContains(t, result.Err, `failed to detect migration state of dogu "nexus"`)
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
		assert.False(t, result.Continue)
	})
}
