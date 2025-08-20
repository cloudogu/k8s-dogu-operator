package controllers

import (
	"context"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const requeueAfterSensitiveConfigOwnerReference = 5 * time.Second

type SensitiveConfigOwnerReferenceStep struct {
	doguConfigRepository doguConfigRepository
}

func NewSensitiveConfigOwnerReferenceStep(configRepos util.ConfigRepositories) *SensitiveConfigOwnerReferenceStep {
	return &SensitiveConfigOwnerReferenceStep{
		doguConfigRepository: configRepos.SensitiveDoguRepository,
	}
}

func (dcs *SensitiveConfigOwnerReferenceStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	err = dcs.doguConfigRepository.SetOwnerReference(ctx, cescommons.SimpleName(doguResource.Name), []metav1.OwnerReference{
		{
			Name:       doguResource.Name,
			Kind:       doguResource.Kind,
			APIVersion: doguResource.APIVersion,
			UID:        doguResource.UID,
		},
	})
	if err != nil {
		return requeueAfterSensitiveConfigOwnerReference, err
	}

	return 0, nil
}
