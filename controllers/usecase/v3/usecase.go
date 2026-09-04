package v2

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v3/install"
)

type DoguUseCase struct {
	steps []v3.Step
}

func NewDoguDeleteUseCase() *DoguUseCase {
	return &DoguUseCase{
		steps: []v3.Step{}}
}

//nolint:funlen
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
