package async

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal"
)

const FinishedState = "finished"

type asyncExecutionController struct {
	steps []internal.AsyncStep
}

// NewAsyncExecutionController creates a new instance of asyncExecutionController.
func NewAsyncExecutionController() *asyncExecutionController {
	return &asyncExecutionController{}
}

// AddStep adds a step.
func (s *asyncExecutionController) AddStep(step internal.AsyncStep) {
	s.steps = append(s.steps, step)
}

// Execute executes all steps.
func (s *asyncExecutionController) Execute(ctx context.Context, dogu *k8sv1.Dogu, currentState string) error {
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
