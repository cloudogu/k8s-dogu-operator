package resource

import (
	"fmt"

	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// applier provides ways to apply unstructured Kubernetes resources against the API.
type applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type collectApplier struct {
	applier applier
}

// NewCollectApplier creates a K8s resource applier that filters and collects deployment resources for a later,
// customized application.
func NewCollectApplier(applier applier) *collectApplier {
	return &collectApplier{applier: applier}
}

// CollectApply applies the given resource but filters deployment resources and returns them so that they can be
// applied later.
func (ca *collectApplier) CollectApply(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	if len(customK8sResources) == 0 {
		logger.Info("No custom K8s resources found")
		return nil, nil
	}

	targetNamespace := doguResource.ObjectMeta.Namespace

	namespaceTemplate := struct {
		Namespace string
	}{
		Namespace: targetNamespace,
	}

	dCollector := &deploymentCollector{collected: []*appsv1.Deployment{}}

	for file, yamlDocs := range customK8sResources {
		logger.Info(fmt.Sprintf("Applying custom K8s resources from file %s", file))

		err := apply.NewBuilder(ca.applier).
			WithNamespace(targetNamespace).
			WithOwner(doguResource).
			WithTemplate(file, namespaceTemplate).
			WithCollector(dCollector).
			WithYamlResource(file, []byte(yamlDocs)).
			WithApplyFilter(&deploymentAntiFilter{}).
			ExecuteApply()

		if err != nil {
			return nil, err
		}
	}

	if len(dCollector.collected) > 1 {
		return nil, fmt.Errorf("expected exactly one Deployment but found %d - not sure how to continue", len(dCollector.collected))
	}
	if len(dCollector.collected) == 1 {
		return dCollector.collected[0], nil
	}

	return nil, nil

}

type deploymentCollector struct {
	collected []*appsv1.Deployment
}

func (dc *deploymentCollector) Predicate(doc apply.YamlDocument) (bool, error) {
	var deployment = &appsv1.Deployment{}

	err := yaml.Unmarshal(doc, deployment)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal object [%s] into deployment: %w", string(doc), err)
	}

	return deployment.Kind == "Deployment", nil
}

func (dc *deploymentCollector) Collect(doc apply.YamlDocument) {
	var deployment = &appsv1.Deployment{}

	// ignore error because it has already been parsed in Predicate()
	_ = yaml.Unmarshal(doc, deployment)

	dc.collected = append(dc.collected, deployment)
}

type deploymentAntiFilter struct{}

func (dc *deploymentAntiFilter) Predicate(doc apply.YamlDocument) (bool, error) {
	var deployment = &appsv1.Deployment{}

	err := yaml.Unmarshal(doc, deployment)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal object [%s] into deployment: %w", string(doc), err)
	}

	return deployment.Kind != "Deployment", nil
}
