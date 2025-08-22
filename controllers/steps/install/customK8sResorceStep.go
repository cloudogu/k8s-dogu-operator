package install

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterCustomK8sResourceStep = 10 * time.Second

type customK8sResourceStep struct {
}

func NewCustomK8sResourceStep(mgrSet util.ManagerSet) *customK8sResourceStep {
	return &customK8sResourceStep{}
}

func (ses *customK8sResourceStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	// TODO
	return 0, nil
}
