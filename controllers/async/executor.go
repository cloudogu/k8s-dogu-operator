package async

import (
	"context"
	"fmt"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// FinishedState is the state where the executor will finish his execution.
const FinishedState = "finished"

type doguExecutionController struct {
	steps []AsyncStep
}

// NewDoguExecutionController creates a new instance of doguExecutionController.
func NewDoguExecutionController() *doguExecutionController {
	return &doguExecutionController{}
}

// AddStep adds a step.
func (s *doguExecutionController) AddStep(step AsyncStep) {
	s.steps = append(s.steps, step)
}

// Execute executes all steps.
func (s *doguExecutionController) Execute(ctx context.Context, dogu *k8sv2.Dogu, currentState string) error {
	if currentState == FinishedState {
		return nil
	}

	for _, step := range s.steps {
		if currentState == step.GetStartCondition() {
			nextState, err := step.Execute(ctx, dogu)
			if err != nil {
				return err
			}

			return s.Execute(ctx, dogu, nextState)
		}
		continue
	}

	return fmt.Errorf("failed to find state in step list")
}
