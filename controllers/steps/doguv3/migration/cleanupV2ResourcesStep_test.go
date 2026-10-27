package migration

import (
	"testing"

	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/stretchr/testify/assert"
)

func TestCleanupV2ResourcesStep_Run(t *testing.T) {
	recorder := NewMockEventRecorder(t)
	step := NewCleanupV2ResourcesStep(recorder)

	result := step.Run(testCtx, testDogu())

	assert.Equal(t, stepsv3.Continue(), result)
}
