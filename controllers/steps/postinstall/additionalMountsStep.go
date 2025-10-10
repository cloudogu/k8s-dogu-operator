package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The AdditionalMountsStep updates the additional mounts if they have changed.
type AdditionalMountsStep struct {
	additionalMountManager
}

func NewAdditionalMountsStep(mountManager manager.AdditionalMountManager) *AdditionalMountsStep {
	return &AdditionalMountsStep{
		additionalMountManager: mountManager,
	}
}

func (ams *AdditionalMountsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	changed, err := ams.AdditionalMountsChanged(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if changed {
		err = ams.UpdateAdditionalMounts(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}
	return steps.Continue()
}
