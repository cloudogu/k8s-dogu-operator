package v3

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3/install"
)

// Usecases v3 and v2 could be refactored with go1.27 generics

// DoguUseCase is a controlling structure responsible for handling all those actions that occur within the dogu's lifecycle phase. The actions are set up as [v3.Step]s during the use-case construction, independently of the dogu at hand, i. e. different dogus within the same lifecycle phase traverse the same list of steps.
type DoguUseCase struct {
	steps []v3.Step
}

func NewDoguDeleteUseCase() *DoguUseCase {
	return &DoguUseCase{
		steps: []v3.Step{}}
}

func NewDoguInstallOrChangeUseCase(dummyStep *install.DummyStep) *DoguUseCase {
	return &DoguUseCase{
		steps: []v3.Step{
			dummyStep,
		},
	}
}

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
