package exec

import (
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type k8sClient interface {
	client.Client
}

type remoteExecutor interface {
	remotecommand.Executor
}
