package nginx

import "sigs.k8s.io/controller-runtime/pkg/client"

type K8sClient interface {
	client.Client
}
