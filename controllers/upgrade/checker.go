package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
)

type checker struct {
	localDoguFetcher localDoguFetcher
}

func NewChecker(localDoguFetcher cesregistry.LocalDoguFetcher) Checker {
	return &checker{localDoguFetcher: localDoguFetcher}
}

// IsUpgrade returns if a dogu needs to be upgraded
func (c *checker) IsUpgrade(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	doguDescriptor, err := c.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return false, fmt.Errorf("failed to fetch dogu when checking for upgrade: %w", err)
	}

	desiredVersion, err := core.ParseVersion(doguResource.Spec.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse desired dogu version: %w", err)
	}

	installedVersion, err := core.ParseVersion(doguDescriptor.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse installed dogu version: %w", err)
	}

	return desiredVersion.IsNewerThan(installedVersion), nil
}
