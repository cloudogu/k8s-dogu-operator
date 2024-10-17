package nginx

import "sigs.k8s.io/controller-runtime/pkg/client"

//nolint:unused
//goland:noinspection GoUnusedType
type k8sClient interface {
	client.Client
}
