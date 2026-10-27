package migration

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type V2ResourceFinder struct {
	client client.Reader
}

func NewV2ResourceFinder(client client.Reader) V2ResourceFinder {
	return V2ResourceFinder{
		client: client,
	}
}

func (vrf V2ResourceFinder) FindV2Resources(ctx context.Context, doguName string) ([]client.Object, error) {
	return nil, nil
}
