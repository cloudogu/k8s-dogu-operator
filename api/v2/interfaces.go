package v2

import "sigs.k8s.io/controller-runtime/pkg/client"

// +kubebuilder:object:generate:=false
type K8sClient interface {
	client.Client
}
