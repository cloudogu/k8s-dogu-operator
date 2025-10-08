package postinstall

import (
	"context"
	"fmt"
	"github.com/cloudogu/retry-lib/retry"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type SecurityContextStep struct {
	localDoguFetcher         localDoguFetcher
	securityContextGenerator securityContextGenerator
	deploymentInterface      deploymentInterface
}

func NewSecurityContextStep(fetcher cesregistry.LocalDoguFetcher, generator resource.SecurityContextGenerator, deploymentInterface v1.DeploymentInterface) *SecurityContextStep {
	return &SecurityContextStep{
		localDoguFetcher:         fetcher,
		securityContextGenerator: generator,
		deploymentInterface:      deploymentInterface,
	}
}

func (scs *SecurityContextStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := scs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err))
	}

	err = retry.OnConflict(func() error {
		var retryErr error
		deployment, retryErr := scs.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
		if retryErr != nil {
			return fmt.Errorf("failed to get deployment of dogu %q: %w", doguResource.Name, retryErr)
		}

		podSecurityContext, containerSecurityContext := scs.securityContextGenerator.Generate(ctx, dogu, doguResource)

		deployment.Spec.Template.Spec.SecurityContext = podSecurityContext
		for i := range deployment.Spec.Template.Spec.Containers {
			deployment.Spec.Template.Spec.Containers[i].SecurityContext = containerSecurityContext
		}

		_, retryErr = scs.deploymentInterface.Update(ctx, deployment, metav1.UpdateOptions{})
		return retryErr
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
