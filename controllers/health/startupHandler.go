package health

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type StartupHandler struct {
	doguInterface           cloudogu.DoguInterface
	deploymentInterface     v1.DeploymentInterface
	availabilityChecker     cloudogu.DeploymentAvailabilityChecker
	doguHealthStatusUpdater cloudogu.DoguHealthStatusUpdater
}

func NewStartupHandler(doguInterface cloudogu.DoguInterface, deploymentInterface v1.DeploymentInterface,
	availabilityChecker cloudogu.DeploymentAvailabilityChecker, healthUpdater cloudogu.DoguHealthStatusUpdater) *StartupHandler {
	return &StartupHandler{
		doguInterface:           doguInterface,
		deploymentInterface:     deploymentInterface,
		availabilityChecker:     availabilityChecker,
		doguHealthStatusUpdater: healthUpdater,
	}
}

func (s *StartupHandler) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.WithName("health startup handler").Info("updating health of all dogus on startup")

	list, err := s.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var errs []error
	for _, dogu := range list.Items {
		var statusErr error
		deployment, deployErr := s.deploymentInterface.Get(ctx, dogu.Name, metav1.GetOptions{})
		if deployErr != nil {
			if apierrors.IsNotFound(deployErr) {
				logger.Error(deployErr, fmt.Sprintf("no deployment found for dogu: %s", dogu.Name))
			} else {
				errs = append(errs, fmt.Errorf("failed to get deployment %q: %w", dogu.Name, deployErr))
			}
			statusErr = s.doguHealthStatusUpdater.UpdateStatus(ctx, types.NamespacedName{Name: dogu.Name, Namespace: dogu.Namespace}, false)
		} else {
			doguAvailable := s.availabilityChecker.IsAvailable(deployment)
			statusErr = s.doguHealthStatusUpdater.UpdateStatus(ctx, types.NamespacedName{Name: dogu.Name, Namespace: dogu.Namespace}, doguAvailable)
		}
		if statusErr != nil {
			errs = append(errs, fmt.Errorf("failed to refresh health status of %q: %w", dogu.Name, statusErr))
		}
	}
	return errors.Join(errs...)
}
