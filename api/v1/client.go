package v1

const (
	// DoguServiceAccountKind describes a service account on a dogu.
	DoguServiceAccountKind ServiceAccountKind = "dogu"
	// KubernetesServiceAccountKind describes a service account on kubernetes.
	KubernetesServiceAccountKind ServiceAccountKind = "k8s"
)

// DoguOperatorClient references the k8s-dogu-operator as a client for the creation of a resource.
const DoguOperatorClient = "k8s-dogu-operator"

// ConfigMapParamType describes a volume of type config map.
const ConfigMapParamType VolumeParamsType = "configmap"

// VolumeParamsType describes the kind of volume the k8s-dogu-operator should create.
type VolumeParamsType string

// VolumeParams contains additional information for the k8s-dogu-operator to create a volume.
type VolumeParams struct {
	// Type describes the kind of volume the k8s-dogu-operator should create.
	Type VolumeParamsType
	// Content contains the actual information that is needed to create a volume of a given Type.
	// The structure of this information is therefore dependent on the Type.
	// To describe a configmap, it could f.i. contain data of type VolumeConfigMapContent.
	Content interface{}
}

// VolumeConfigMapContent contains information needed to create a volume of type configmap.
type VolumeConfigMapContent struct {
	// Name of the configmap to create.
	Name string
}

// ServiceAccountKind defines the kind of service on which the account should be created.
type ServiceAccountKind string
