package resource

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	v1 "k8s.io/api/core/v1"
)

type globalConfigurationWatcher interface {
	// Watch watches for changes of the provided config-key and sends the event through the channel
	Watch(ctx context.Context, filters ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error)
}

// RequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type requirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (v1.ResourceRequirements, error)
}

// hostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
type hostAliasGenerator interface {
	Generate(context.Context) (hostAliases []v1.HostAlias, err error)
}

type doguConfigGetter interface {
	Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error)
}

type doguGetter interface {
	GetCurrent(ctx context.Context, simpleDoguName string) (*cesappcore.Dogu, error)
}
