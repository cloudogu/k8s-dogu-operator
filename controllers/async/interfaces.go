package async

import (
	"context"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

// AsyncStep capsules an action with a starting and end condition
type AsyncStep interface {
	// GetStartCondition returns the start condition for the step.
	GetStartCondition() string
	// Execute executes the step and returns the end condition of the step.
	Execute(ctx context.Context, dogu *k8sv2.Dogu) (string, error)
}

// AsyncExecutor collects steps and executes them all.
type AsyncExecutor interface {
	// AddStep adds a step.
	AddStep(step AsyncStep)
	// Execute executes all steps.
	Execute(ctx context.Context, dogu *k8sv2.Dogu, currentState string) error
}
