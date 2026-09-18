package doguv3

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
)

// DoguUseCase is a controlling structure responsible for handling all those actions that occur within the dogu's lifecycle phase.
// The actions are set up as [v3.Step]s during the use-case construction, independently of the dogu at hand, i. e. different
// dogus within the same lifecycle phase traverse the same list of steps.
type DoguUseCase struct {
	steps []v3.Step
}

func NewDoguDeleteUseCase() *DoguUseCase {
	return &DoguUseCase{
		steps: []v3.Step{}}
}

// NewDoguInstallOrChangeUseCase creates a new DoguUseCase instance with the given steps.
// The steps are executed in the order they are provided.
// With new steps the parameter list will grow.
// Unfortunately, this is the only way to provide the steps in the correct order.
// Uber fx provides value groups to use variadic parameters like NewDoguInstallOrChangeUseCase(steps ...doguv3.Step),
// but these are unordered.
func NewDoguInstallOrChangeUseCase(dummyStep *install.DummyStep) *DoguUseCase {
	return &DoguUseCase{
		steps: []v3.Step{
			dummyStep,
		},
	}
}

// HandleUntilApplied acts as the core control function which runs all use-case steps, independently of the dogu at hand. This function will wait until the Step was fully executed, resulting in either
//   - continuing to the next step (if any),
//   - a requeue if a step needs more time or errored in a way that might be handled programmatically
//   - an abortion of the dogus reconciliation
//
// If the dogu executed all steps successfully, the duration and the error will contain null values and the bool is set to true indicating the dogu phase is done.
func (duc *DoguUseCase) HandleUntilApplied(ctx context.Context, doguResource *v3beta1.Dogu) (time.Duration, bool, error) {
	for _, s := range duc.steps {
		result := s.Run(ctx, doguResource)
		if result.Err != nil || result.RequeueAfter != 0 {
			return result.RequeueAfter, false, result.Err
		}
		if !result.Continue {
			return 0, false, nil
		}
	}
	return 0, true, nil
}
