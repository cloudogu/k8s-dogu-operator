package upgrade

import (
	"context"
	"errors"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

const (
	serviceAccountKindDogu    = "dogu"
	serviceAccountKindDefault = ""
)

// RetroactiveServiceAccountStep issues reconcile events for dogus that define a service account for the currently reconciled dogu.
// The service account will then be created by install.ServiceAccountStep if it does not exist.
// This ensures that e.g. service accounts for optional dependencies get created retroactively when that dependency gets installed.
type RetroactiveServiceAccountStep struct {
	doguEvents       chan<- event.TypedGenericEvent[*doguv2.Dogu]
	doguClient       doguInterface
	localDoguFetcher localDoguFetcher
}

func NewRetroactiveServiceAccountStep(
	doguEvents chan<- event.TypedGenericEvent[*doguv2.Dogu],
	doguClient doguClient.DoguInterface,
	localDoguFetcher cesregistry.LocalDoguFetcher,
) *RetroactiveServiceAccountStep {
	return &RetroactiveServiceAccountStep{
		doguEvents:       doguEvents,
		doguClient:       doguClient,
		localDoguFetcher: localDoguFetcher,
	}
}

func (r *RetroactiveServiceAccountStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	doguList, err := r.doguClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("list dogus for retroactive service accounts: %w", err))
	}

	var errs []error
	for _, dogu := range doguList.Items {
		doguDescriptor, err := r.localDoguFetcher.FetchForResource(ctx, &dogu)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, serviceAccount := range doguDescriptor.ServiceAccounts {
			if (serviceAccount.Kind == serviceAccountKindDogu ||
				serviceAccount.Kind == serviceAccountKindDefault) &&
				serviceAccount.Type == resource.Name {
				r.doguEvents <- event.TypedGenericEvent[*doguv2.Dogu]{Object: &dogu}
				break // only one reconcile necessary per dogu
			}
		}
	}

	if len(errs) > 0 {
		return steps.RequeueWithError(fmt.Errorf("retrieve retroactive service accounts: %w", errors.Join(errs...)))
	}

	return steps.Continue()
}
