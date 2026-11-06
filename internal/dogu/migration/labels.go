package migration

import (
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
)

// migrationLabels identify the temporary resources of a dogu's migration.
func migrationLabels(dogu *v3beta1.Dogu) map[string]string {
	return map[string]string{
		resource.CloudoguLabelKeyApp: resource.LabelValueCes,
		v3beta1.DoguLabelName:        dogu.Spec.Name,
		configTypeLabel:              migrationType,
	}
}
