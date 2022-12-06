package resource

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type collectApplier struct {
	applier internal.Applier
}

// NewCollectApplier creates a K8s resource applier that filters and collects deployment resources for a later,
// customized application.
func NewCollectApplier(applier internal.Applier) *collectApplier {
	return &collectApplier{applier: applier}
}

// CollectApply applies the given resource.
func (ca *collectApplier) CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) error {
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
