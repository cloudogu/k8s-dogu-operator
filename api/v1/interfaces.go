package v1

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sClient interface {
	client.Client
}

type k8sSubResourceWriter interface {
	client.SubResourceWriter
}
