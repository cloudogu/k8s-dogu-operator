package upgrade

import (
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type doguFetcher struct {
	doguLocalRegistry  registry.DoguRegistry
	doguRemoteRegistry cesremote.Registry
}

// NewDoguFetcher creates a new dogu fetcher that provides descriptors for the currently installed dogu and the remote
// dogu being used as upgrade.
func NewDoguFetcher(doguLocalRegistry registry.DoguRegistry, doguRemoteRegistry cesremote.Registry) *doguFetcher {
	return &doguFetcher{doguLocalRegistry: doguLocalRegistry, doguRemoteRegistry: doguRemoteRegistry}
}

func (df *doguFetcher) Fetch(toDoguResource *k8sv1.Dogu) (fromDogu *core.Dogu, toDogu *core.Dogu, err error) {
	localDoguName := toDoguResource.Name
	upgradeDoguName := toDoguResource.Spec.Name
	upgradeDoguVersion := toDoguResource.Spec.Version

	fromDogu, err = df.getLocalDogu(toDoguResource.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get local dogu descriptor for %s: %w", localDoguName, err)
	}

	toDogu, err = df.getRemoteDogu(upgradeDoguName, upgradeDoguVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get remote dogu descriptor for %s:%s: %w", upgradeDoguName, upgradeDoguVersion, err)
	}
	return fromDogu, toDogu, nil
}

func (df *doguFetcher) getLocalDogu(fromDoguName string) (*core.Dogu, error) {
	return df.doguLocalRegistry.Get(fromDoguName)
}

func (df *doguFetcher) getRemoteDogu(name, version string) (*core.Dogu, error) {
	return df.doguRemoteRegistry.GetVersion(name, version)
}
