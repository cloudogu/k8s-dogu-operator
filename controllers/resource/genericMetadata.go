package resource

// Dogu v3 label keys and generic values. Some of them are label names commonly recommended by Kubernetes
// (see https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/#labels), hence their
// names are prefixed with "Common" here.
const (
	// CommonLabelKeyVersion contains the label key to the tool's chart (or if not applicable: the application) version.
	CommonLabelKeyVersion = "app.kubernetes.io/version"
	// CommonLabelKeyName contains the label key to the tool's simple name.
	CommonLabelKeyName = "app.kubernetes.io/name"
	// CommonLabelKeyPartOf contains the label key to the tool suit's simple name.
	CommonLabelKeyPartOf = "app.kubernetes.io/part-of"
	// CloudoguLabelKeyApp is a more updated label key than LegacyLabelKeyApp but content-wise the same.
	CloudoguLabelKeyApp = "k8s.cloudogu.com/app"
	// LegacyLabelKeyApp contains the label key to the general app name in the legacy form.
	// For Cloudogu controlled applications that is usually LabelValueCes.
	LegacyLabelKeyApp = "app"

	// LabelValueCes contains the generic CES label, which is used as a value for labels CommonLabelKeyPartOf,
	// CloudoguLabelKeyApp, or LegacyLabelKeyApp.
	LabelValueCes = "ces"
)
