package resource

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
)

type collectApplier struct {
	applier Applier
}

// NewCollectApplier creates a K8s resource applier that filters and collects deployment resources for a later,
// customized application.
func NewCollectApplier(applier Applier) *collectApplier {
	return &collectApplier{applier: applier}
}

// CollectApply applies the given resource.
func (ca *collectApplier) CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv2.Dogu) error {
	if len(customK8sResources) == 0 {
		logging.Info(ctx, "No custom K8s resources found")
		return nil
	}

	targetNamespace := doguResource.Namespace

	namespaceTemplate := struct {
		Namespace string
	}{
		Namespace: targetNamespace,
	}

	for file, yamlDocs := range customK8sResources {
		logging.Info(ctx, fmt.Sprintf("Applying custom K8s resources from file %s", file))

		err := apply.NewBuilder(ca.applier).
			WithNamespace(targetNamespace).
			WithOwner(doguResource).
			WithTemplate(file, namespaceTemplate).
			WithYamlResource(file, []byte(yamlDocs)).
			ExecuteApply()

		if err != nil {
			return err
		}
	}

	return nil

}
