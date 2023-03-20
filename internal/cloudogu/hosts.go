package cloudogu

import v1 "k8s.io/api/core/v1"

type HostAliasGenerator interface {
	Generate() ([]v1.HostAlias, error)
}
