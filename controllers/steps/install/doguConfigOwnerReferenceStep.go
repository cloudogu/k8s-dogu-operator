package install

import (
	"context"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const requeueAfterDoguConfigOwnerReference = 5 * time.Second

type DoguConfigOwnerReferenceStep struct {
	doguConfigRepository doguConfigRepository
}

func NewDoguConfigOwnerReferenceStep(configRepos util.ConfigRepositories) *DoguConfigOwnerReferenceStep {
	return &DoguConfigOwnerReferenceStep{
		doguConfigRepository: configRepos.DoguConfigRepository,
	}
}

func (dcs *DoguConfigOwnerReferenceStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	err = dcs.doguConfigRepository.SetOwnerReference(ctx, cescommons.SimpleName(doguResource.Name), []metav1.OwnerReference{
		{
			Name:       doguResource.Name,
			Kind:       doguResource.Kind,
			APIVersion: doguResource.APIVersion,
			UID:        doguResource.UID,
		},
	})
	if err != nil {
		return 0, err
	}

	return 0, nil
}
