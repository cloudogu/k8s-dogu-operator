package initfx

import (
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-host-change/pkg/alias"
)

func NewHostAliasGenerator(repository resource.GlobalConfigRepository) resource.HostAliasGenerator {
	return alias.NewHostAliasGenerator(repository)
}
