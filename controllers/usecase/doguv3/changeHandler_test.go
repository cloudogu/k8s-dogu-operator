package doguv3

import (
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// requeueingUseCase returns a use-case whose only step requeues after the given duration.
func requeueingUseCase(t *testing.T, doguResource *v3beta1.Dogu, requeueAfter time.Duration) *DoguUseCase {
	step := NewMockStep(t)
	step.EXPECT().Run(mock.Anything, doguResource).Return(doguv3.RequeueAfter(requeueAfter, "Testing", "requeue"))
	return useCaseWithSteps(doguResource, step)
}

// unusedUseCase returns a use-case whose step fails the test when it runs.
func unusedUseCase(t *testing.T, doguResource *v3beta1.Dogu) *DoguUseCase {
	return useCaseWithSteps(doguResource, NewMockStep(t))
}

func useCaseWithSteps(doguResource *v3beta1.Dogu, steps ...Step) *DoguUseCase {
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(doguResource).WithStatusSubresource(&v3beta1.Dogu{}).Build()
	return &DoguUseCase{steps: steps, k8sClient: client}
}

func TestDoguChangeHandler_HandleUntilApplied(t *testing.T) {
	t.Run("dogu without migration is installed or changed", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, doguResource).Return(migration.NotRequired, nil)
		handler := NewDoguChangeHandler(detector, requeueingUseCase(t, doguResource, 5*time.Second), unusedUseCase(t, doguResource))

		requeueAfter, err := handler.HandleUntilApplied(testCtx, doguResource)

		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, requeueAfter)
	})

	t.Run("dogu with required migration is migrated", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, doguResource).Return(migration.Required, nil)
		handler := NewDoguChangeHandler(detector, unusedUseCase(t, doguResource), requeueingUseCase(t, doguResource, 2*time.Minute))

		requeueAfter, err := handler.HandleUntilApplied(testCtx, doguResource)

		assert.NoError(t, err)
		assert.Equal(t, 2*time.Minute, requeueAfter)
	})

	t.Run("dogu with migration in progress is migrated", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, doguResource).Return(migration.InProgress, nil)
		handler := NewDoguChangeHandler(detector, unusedUseCase(t, doguResource), requeueingUseCase(t, doguResource, 7*time.Minute))

		requeueAfter, err := handler.HandleUntilApplied(testCtx, doguResource)

		assert.NoError(t, err)
		assert.Equal(t, 7*time.Minute, requeueAfter)
	})

	t.Run("detection error is returned without running a use-case", func(t *testing.T) {
		doguResource := testDoguResource.DeepCopy()
		detector := NewMockMigrationDetector(t)
		detector.EXPECT().Detect(testCtx, doguResource).Return(migration.NotRequired, assert.AnError)
		handler := NewDoguChangeHandler(detector, unusedUseCase(t, doguResource), unusedUseCase(t, doguResource))

		requeueAfter, err := handler.HandleUntilApplied(testCtx, doguResource)

		assert.Zero(t, requeueAfter)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, `failed to detect migration state of dogu "nexus"`)
	})
}
