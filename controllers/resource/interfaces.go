package resource

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sClient interface {
	client.Client
}
