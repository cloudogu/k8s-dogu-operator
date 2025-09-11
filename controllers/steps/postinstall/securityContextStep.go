package postinstall

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecurityContextStep struct {
	localDoguFetcher         localDoguFetcher
	securityContextGenerator securityContextGenerator
	deploymentInterface      deploymentInterface
}

func NewSecurityContextStep(
	mgrSet *util.ManagerSet,
	namespace string,
) *SecurityContextStep {
	return &SecurityContextStep{
		localDoguFetcher:         mgrSet.LocalDoguFetcher,
		securityContextGenerator: resource.NewSecurityContextGenerator(),
		deploymentInterface:      mgrSet.ClientSet.AppsV1().Deployments(namespace),
	}
}

func (scs *SecurityContextStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := scs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err))
	}

	deployment, err := scs.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get deployment of dogu %q: %w", doguResource.Name, err))
	}

	podSecurityContext, containerSecurityContext := scs.securityContextGenerator.Generate(ctx, dogu, doguResource)

	deployment.Spec.Template.Spec.SecurityContext = podSecurityContext
	for i := range deployment.Spec.Template.Spec.Containers {
		deployment.Spec.Template.Spec.Containers[i].SecurityContext = containerSecurityContext
	}

	_, err = scs.deploymentInterface.Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
