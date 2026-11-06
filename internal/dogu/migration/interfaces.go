package migration

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ListReader interface {
	client.Reader
}
