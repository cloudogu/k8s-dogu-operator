package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type ServiceStep struct {
	serviceGenerator serviceGenerator
	localDoguFetcher localDoguFetcher
	imageRegistry    imageRegistry
	serviceInterface serviceInterface
}

func NewServiceStep(registry imageregistry.ImageRegistry, serviceInterface v1.ServiceInterface, generator resource.DoguResourceGenerator, fetcher cesregistry.LocalDoguFetcher) *ServiceStep {
	return &ServiceStep{
		imageRegistry:    registry,
		serviceInterface: serviceInterface,
		serviceGenerator: generator,
		localDoguFetcher: fetcher,
	}
}

func (ses *ServiceStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := ses.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	imageConfig, err := ses.imageRegistry.PullImageConfig(ctx, doguDescriptor.Image+":"+doguDescriptor.Version)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	service, err := ses.serviceGenerator.CreateDoguService(doguResource, doguDescriptor, imageConfig)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = ses.createOrUpdateService(ctx, service)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}

func (ses *ServiceStep) createOrUpdateService(ctx context.Context, service *corev1.Service) error {
	_, err := ses.serviceInterface.Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}

		_, err = ses.serviceInterface.Create(ctx, service, metav1.CreateOptions{})
		return err
	}

	_, err = ses.serviceInterface.Update(ctx, service, metav1.UpdateOptions{})
	return err
}
