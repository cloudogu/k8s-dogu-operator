package v2

import "sigs.k8s.io/controller-runtime/pkg/client"

// +kubebuilder:object:generate:=false
//
//nolint:unused
//goland:noinspection GoUnusedType
type k8sClient interface {
	client.Client
}
