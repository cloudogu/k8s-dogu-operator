package postinstall

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecurityContextStep struct {
	localDoguFetcher         localDoguFetcher
	securityContextGenerator resource.SecurityContextGenerator
	deploymentInterface      deploymentInterface
}

func NewSecurityContextStep(
	fetcher localDoguFetcher,
	generator resource.SecurityContextGenerator,
	deplInt deploymentInterface,
) *SecurityContextStep {
	return &SecurityContextStep{
		localDoguFetcher:         fetcher,
		securityContextGenerator: generator,
		deploymentInterface:      deplInt,
	}
}

func (scs *SecurityContextStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	dogu, err := scs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return 0, fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err)
	}

	deployment, err := scs.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get deployment of dogu %q: %w", doguResource.Name, err)
	}

	podSecurityContext, containerSecurityContext := scs.securityContextGenerator.Generate(ctx, dogu, doguResource)

	deployment.Spec.Template.Spec.SecurityContext = podSecurityContext
	for i := range deployment.Spec.Template.Spec.Containers {
		deployment.Spec.Template.Spec.Containers[i].SecurityContext = containerSecurityContext
	}

	_, err = scs.deploymentInterface.Update(ctx, deployment, metav1.UpdateOptions{})
	return 0, err
}
