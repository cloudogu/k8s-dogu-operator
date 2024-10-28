package resource

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"

	"sigs.k8s.io/controller-runtime/pkg/log"
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
	logger := log.FromContext(ctx)
	if len(customK8sResources) == 0 {
		logger.Info("No custom K8s resources found")
		return nil
	}

	targetNamespace := doguResource.ObjectMeta.Namespace

	namespaceTemplate := struct {
		Namespace string
	}{
		Namespace: targetNamespace,
	}

	for file, yamlDocs := range customK8sResources {
		logger.Info(fmt.Sprintf("Applying custom K8s resources from file %s", file))

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
