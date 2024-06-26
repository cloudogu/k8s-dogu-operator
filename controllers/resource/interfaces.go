package resource

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/registry"
	v1 "k8s.io/api/core/v1"
)

type GlobalConfigurationWatcher interface {
	// Watch watches for changes of the provided config-key and sends the event through the channel
	Watch(ctx context.Context, key string, recursive bool) (registry.ConfigWatch, error)
}

// RequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type RequirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (v1.ResourceRequirements, error)
}

type DoguConfigRegistry interface {
	DoguConfigValueGetter
}

type DoguConfigProvider interface {
	GetDoguConfig(ctx context.Context, doguName string) (DoguConfigRegistry, error)
}

type DoguConfigValueGetter interface {
	Get(ctx context.Context, key string) (string, error)
}

type DoguGetter interface {
	GetCurrent(ctx context.Context, simpleDoguName string) (*cesappcore.Dogu, error)
}

// HostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
type HostAliasGenerator interface {
	Generate() (hostAliases []v1.HostAlias, err error)
}
