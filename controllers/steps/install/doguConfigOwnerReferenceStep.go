package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DoguConfigOwnerReferenceStep struct {
	doguConfigRepository doguConfigRepository
}

func NewDoguConfigOwnerReferenceStep(configRepos util.ConfigRepositories) *DoguConfigOwnerReferenceStep {
	return &DoguConfigOwnerReferenceStep{
		doguConfigRepository: configRepos.DoguConfigRepository,
	}
}

func (dcs *DoguConfigOwnerReferenceStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := dcs.doguConfigRepository.SetOwnerReference(ctx, cescommons.SimpleName(doguResource.Name), []metav1.OwnerReference{
		{
			Name:       doguResource.Name,
			Kind:       doguResource.Kind,
			APIVersion: doguResource.APIVersion,
			UID:        doguResource.UID,
		},
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
