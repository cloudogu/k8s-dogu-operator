package registry

import (
	dccv3 "github.com/cloudogu/dogu-lib/doguv3/doguregistry/dcc/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

// NewDoguRegistryReader builds the v3 dogu registry reader (a dcc HTTP client) from the operator's
// remote configuration and credentials.
func NewDoguRegistryReader(operatorConfig *config.OperatorConfig) (*dccv3.DccHttpClient, error) {
	configuration, err := operatorConfig.GetV3RemoteConfiguration()
	if err != nil {
		return nil, err
	}

	client, err := dccv3.New(configuration, operatorConfig.GetV3RemoteCredentials())
	if err != nil {
		return nil, err
	}

	return client, nil
}
