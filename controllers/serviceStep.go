package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const requeueAfterService = 10 * time.Second

type ServiceStep struct {
	serviceGenerator    serviceGenerator
	resourceDoguFetcher resourceDoguFetcher
	imageRegistry       imageRegistry
	serviceInterface    serviceInterface
}

func NewServiceStep(mgrSet util.ManagerSet) *ServiceStep {
	return &ServiceStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
	}
}

func (ses *ServiceStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	doguDescriptor, err := ses.getDoguDescriptor(ctx, doguResource)
	if err != nil {
		return requeueAfterVolume, err
	}
	imageConfig, err := ses.imageRegistry.PullImageConfig(ctx, doguDescriptor.Image+":"+doguResource.Spec.Version)
	service, err := ses.serviceGenerator.CreateDoguService(doguResource, doguDescriptor, imageConfig)
	if err != nil {
		return requeueAfterService, err
	}
	err = ses.createOrUpdateService(ctx, service)
	if err != nil {
		return requeueAfterService, err
	}
	return 0, nil
}

func (ses *ServiceStep) getDoguDescriptor(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	doguDescriptor, _, err := ses.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return doguDescriptor, nil
}

func (ses *ServiceStep) createOrUpdateService(ctx context.Context, service *corev1.Service) error {
	_, err := ses.serviceInterface.Get(ctx, service.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_, err := ses.serviceInterface.Create(ctx, service, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	_, err = ses.serviceInterface.Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
