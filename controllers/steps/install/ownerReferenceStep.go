package install

import (
	"context"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// The OwnerReferenceStep creates an owner reference for the dogu cr on a specific resource.
// It is used for the dogu config and dogu descriptor config maps and secret dogu secrets.
type OwnerReferenceStep struct {
	ownerReferenceSetter ownerReferenceSetter
	doguScheme           *runtime.Scheme
}

func NewOwnerReferenceStep(setter initfx.OwnerReferenceSetter, scheme *runtime.Scheme) *OwnerReferenceStep {
	return &OwnerReferenceStep{
		ownerReferenceSetter: setter,
		doguScheme:           scheme,
	}
}

func (ors *OwnerReferenceStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	gvk, err := apiutil.GVKForObject(doguResource, ors.doguScheme)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = ors.ownerReferenceSetter.SetOwnerReference(ctx, cescommons.SimpleName(doguResource.Name), []metav1.OwnerReference{
		{
			Name:               doguResource.Name,
			Kind:               gvk.Kind,
			APIVersion:         gvk.GroupVersion().String(),
			UID:                doguResource.UID,
			Controller:         &[]bool{true}[0],
			BlockOwnerDeletion: &[]bool{true}[0],
		},
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
