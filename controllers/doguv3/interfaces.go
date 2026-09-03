package doguv3

import "sigs.k8s.io/controller-runtime/pkg/client"

// K8sClient is a generic k8s client using the k8d official interface client.Client which happily caches objects even
// in different occurrences within the whole dogu operator.
type K8sClient interface {
	client.Client
}
