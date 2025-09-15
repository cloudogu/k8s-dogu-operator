package health

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type StartupHandler struct {
	doguInterface doguClient.DoguInterface
	doguEvents    chan<- event.TypedGenericEvent[*v2.Dogu]
}

func NewStartupHandler(manager manager.Manager, doguInterface doguClient.DoguInterface, doguEvents chan<- event.TypedGenericEvent[*v2.Dogu]) (*StartupHandler, error) {
	sh := &StartupHandler{
		doguInterface: doguInterface,
		doguEvents:    doguEvents,
	}
	err := manager.Add(sh)
	if err != nil {
		return nil, err
	}

	return sh, nil
}

func (s *StartupHandler) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.WithName("health startup handler").Info("updating health of all dogus on startup")

	list, err := s.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dogu := range list.Items {
		s.doguEvents <- event.TypedGenericEvent[*v2.Dogu]{Object: &dogu}
	}
	return nil
}
