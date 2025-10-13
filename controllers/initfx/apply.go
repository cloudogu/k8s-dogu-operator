package initfx

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-apply-lib/apply"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"k8s.io/client-go/rest"
)

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

type CollectApplier interface {
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *doguv2.Dogu) error
}

func NewCollectApplier(restConfig *rest.Config) (CollectApplier, error) {
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = doguv2.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}

	collectApplier := resource.NewCollectApplier(applier)
	return collectApplier, nil
}
