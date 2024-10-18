package health

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ShutdownHandler struct {
	doguInterface ecoSystem.DoguInterface
}

func NewShutdownHandler(doguInterface ecoSystem.DoguInterface) *ShutdownHandler {
	return &ShutdownHandler{doguInterface: doguInterface}
}

func (s *ShutdownHandler) Start(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("health shutdown handler")
	<-ctx.Done()
	logger.Info("shutdown detected, handling health status")

	// context is done, we need a new one
	ctx = context.WithoutCancel(ctx)
	return s.handle(ctx)
}

func (s *ShutdownHandler) handle(ctx context.Context) error {
	dogus, err := s.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var errs []error
	for _, dogu := range dogus.Items {
		_, updateErr := s.doguInterface.UpdateStatusWithRetry(ctx, &dogu, func(status v2.DoguStatus) v2.DoguStatus {
			status.Health = v2.UnknownHealthStatus
			return status
		}, metav1.UpdateOptions{})
		if updateErr != nil {
			errs = append(errs, fmt.Errorf("failed to set health status of %q to %q: %w", dogu.Name, v2.UnknownHealthStatus, updateErr))
		}
	}
	return errors.Join(errs...)
}
