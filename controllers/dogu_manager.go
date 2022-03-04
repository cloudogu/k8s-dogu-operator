package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DoguManager struct {
	client.Client
	Scheme            *runtime.Scheme
	resourceGenerator ResourceGenerator
	doguRegistry      DoguRegistry
}

type DoguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*core.Dogu, error)
}

func NewDoguManager(client client.Client, scheme *runtime.Scheme, resourecGenerator ResourceGenerator, doguRegistry DoguRegistry) *DoguManager {
	return &DoguManager{
		Client:            client,
		Scheme:            scheme,
		resourceGenerator: resourecGenerator,
		doguRegistry:      doguRegistry,
	}
}

func (m DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	dogu, err := m.doguRegistry.GetDogu(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	deployment := m.resourceGenerator.GetDoguDeployment(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}

	result, err := ctrl.CreateOrUpdate(ctx, m.Client, deployment, func() error {
		return ctrl.SetControllerReference(doguResource, deployment, m.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}
	logger.Info(fmt.Sprintf("createOrUpdate deployment result: %+v", result))

	service := m.resourceGenerator.GetDoguService(doguResource)
	if err != nil {
		return fmt.Errorf("failed to create dogu service: %w", err)
	}

	result, err = ctrl.CreateOrUpdate(ctx, m.Client, service, func() error {
		return ctrl.SetControllerReference(doguResource, service, m.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to create dogu service: %w", err)
	}
	logger.Info(fmt.Sprintf("createOrUpdate service result: %+v", result))

	return nil
}
