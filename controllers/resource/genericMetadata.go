package resource

// Dogu v3 label keys and generic values
const (
	// LabelKeyAppKubernetesIoVersion contains the label key to the tool's chart (or if not applicable: the application) version.
	LabelKeyAppKubernetesIoVersion = "app.kubernetes.io/version"
	// LabelKeyAppKubernetesIoName contains the label key to the tool's simple name.
	LabelKeyAppKubernetesIoName = "app.kubernetes.io/name"
	// LabelKeyApp contains the label key to the general app name in the legacy form. For Cloudogu controlled applications that is usually LabelValueCes.
	LabelKeyApp = "app"
	// LabelKeyK8sCloudoguComApp is a more updated label key than LabelKeyApp but content-wise the same.
	LabelKeyK8sCloudoguComApp = "k8s.cloudogu.com/app"
	// LabelKeyAppKubernetesIoPartOf contains the label key to the tool suit's simple name.
	LabelKeyAppKubernetesIoPartOf = "app.kubernetes.io/part-of"

	//LabelValueCes contains the generic CES label.
	LabelValueCes = "ces"
)
