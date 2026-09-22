package initfx

import (
	dccv3 "github.com/cloudogu/dogu-lib/doguv3/doguregistry/dcc/client"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v3/client"
	"github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
)

func NewDoguV3Interface(doguClientset doguClient.Interface, config *config.OperatorConfig) v3beta1.DoguInterface {
	return doguClientset.DoguV3beta1().Dogus(config.Namespace)
}

var NewDoguV3RegistryReader = newDoguV3RegistryReader

func newDoguV3RegistryReader(operatorConfig *config.OperatorConfig) (*dccv3.DccHttpClient, error) {
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
