package cesregistry

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	regLibErr "github.com/cloudogu/k8s-registry-lib/errors"
)

func checkDoguVersionEnabled(ctx context.Context, doguVersionRegistry doguVersionRegistry, doguName string) (bool, dogu.DoguVersion, error) {
	currentDoguVersion, err := doguVersionRegistry.GetCurrent(ctx, dogu.SimpleDoguName(doguName))
	if err != nil {
		if regLibErr.IsNotFoundError(err) {
			// no current version found -> not enabled
			return false, currentDoguVersion, nil
		}

		return false, currentDoguVersion, fmt.Errorf("failed to get current version of dogu %s: %w", doguName, err)
	}

	enabled, err := doguVersionRegistry.IsEnabled(ctx, currentDoguVersion)
	if err != nil {
		return false, currentDoguVersion, fmt.Errorf("failed to check if dogu %s is enabled: %w", doguName, err)
	}
	return enabled, currentDoguVersion, nil
}