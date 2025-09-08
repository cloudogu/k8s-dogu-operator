package install

import (
	"context"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DoguDescriptorOwnerReferenceStep struct {
	doguRepository localDoguDescriptorRepository
}

func NewDoguDescriptorOwnerReferenceStep(doguRepository localDoguDescriptorRepository) *DoguDescriptorOwnerReferenceStep {
	return &DoguDescriptorOwnerReferenceStep{
		doguRepository: doguRepository,
	}
}

func (dds *DoguDescriptorOwnerReferenceStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := dds.doguRepository.SetOwnerReference(ctx, cescommons.SimpleName(doguResource.Name), []metav1.OwnerReference{
		{
			Name:       doguResource.Name,
			Kind:       doguResource.Kind,
			APIVersion: doguResource.APIVersion,
			UID:        doguResource.UID,
			Controller: &[]bool{true}[0],
		},
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
