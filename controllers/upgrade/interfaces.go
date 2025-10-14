package upgrade

import (
	"context"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
)

type localDoguFetcher interface {
	cesregistry.LocalDoguFetcher
}

// Checker includes functionality to check for upgrades
type Checker interface {
	// IsUpgrade returns if a dogu needs to be upgraded
	IsUpgrade(ctx context.Context, doguResource *k8sv2.Dogu) (bool, error)
}
